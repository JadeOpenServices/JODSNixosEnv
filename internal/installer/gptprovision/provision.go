// Package gptprovision safely creates the canonical recovery GPT entry while
// preserving the existing operating-system disk layout.
//
// It never recreates the GPT, deletes an existing partition, formats a
// filesystem, changes LUKS metadata, or shrinks storage. If adequate
// unallocated space is unavailable it returns an explicit resize-required
// result for the installer to present to the user.
package gptprovision

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/bakanura/gjallarOS/internal/installer/diskplan"
	"github.com/bakanura/gjallarOS/internal/installer/prompt"
)

const (
	StatusCreated               = "created"
	StatusAlreadyPresent        = "already-present"
	StatusOfflineResizeRequired = "offline-resize-required"
)

var diskGUIDPattern = regexp.MustCompile(
	`(?i)Disk identifier \(GUID\):\s*([0-9a-f-]+)`,
)

type Input struct {
	Plan       diskplan.Plan
	UI         prompt.UI
	Out        io.Writer
	Unattended bool
}

type Result struct {
	Status         string
	PartitionPath  string
	RequiredBytes  uint64
	AvailableBytes uint64
	Message        string
}

type commandRunner interface {
	Run(context.Context, string, ...string) error
	Output(context.Context, string, ...string) ([]byte, error)
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, name string, args ...string) error {
	return exec.CommandContext(ctx, name, args...).Run()
}

func (execRunner) Output(
	ctx context.Context,
	name string,
	args ...string,
) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}

func Provision(ctx context.Context, input Input) (Result, error) {
	return provision(ctx, input, execRunner{})
}

func provision(
	ctx context.Context,
	input Input,
	runner commandRunner,
) (Result, error) {
	if input.Out == nil {
		return Result{}, fmt.Errorf("GPT provisioning output writer is required")
	}

	if err := input.Plan.Validate(); err != nil {
		return Result{}, fmt.Errorf("validate canonical disk plan: %w", err)
	}

	if input.Plan.Recovery == nil {
		return Result{}, fmt.Errorf("canonical plan has no recovery partition")
	}

	before, err := inspectDisk(ctx, runner, input.Plan.TargetDisk.Path)
	if err != nil {
		return Result{}, err
	}

	if err := validateTargetIdentity(input.Plan.TargetDisk, before.Disk); err != nil {
		return Result{}, err
	}

	if !strings.EqualFold(strings.TrimSpace(before.Disk.PTType), "gpt") {
		return Result{}, fmt.Errorf(
			"target disk %s does not use GPT",
			input.Plan.TargetDisk.Path,
		)
	}

	diskGUID, err := readDiskGUID(
		ctx,
		runner,
		input.Plan.TargetDisk.Path,
	)
	if err != nil {
		return Result{}, err
	}

	if !strings.EqualFold(
		diskGUID,
		input.Plan.TargetDisk.GPTDiskGUID,
	) {
		return Result{}, fmt.Errorf(
			"existing GPT disk GUID does not match canonical disk identity",
		)
	}

	recovery := *input.Plan.Recovery

	if existing, found := recoveryPartition(before, recovery); found {
		result := Result{
			Status:        StatusAlreadyPresent,
			PartitionPath: existing.Path,
			RequiredBytes: recovery.SizeBytes,
			Message:       "Canonical recovery partition already exists.",
		}
		fmt.Fprintf(
			input.Out,
			"RECOVERY: canonical partition already exists at %s\n",
			existing.Path,
		)
		return result, nil
	}

	if err := ensureRecoverySlotFree(before, recovery.Number); err != nil {
		return Result{}, err
	}

	sectorSize, err := readUint(
		ctx,
		runner,
		"sudo",
		"blockdev",
		"--getss",
		input.Plan.TargetDisk.Path,
	)
	if err != nil {
		return Result{}, fmt.Errorf("read logical sector size: %w", err)
	}

	if sectorSize == 0 {
		return Result{}, fmt.Errorf("target reports zero logical sector size")
	}

	if recovery.SizeBytes%sectorSize != 0 {
		return Result{}, fmt.Errorf(
			"recovery size %d is not aligned to logical sector size %d",
			recovery.SizeBytes,
			sectorSize,
		)
	}

	freeStart, err := readUint(
		ctx,
		runner,
		"sudo",
		"sgdisk",
		"-F",
		input.Plan.TargetDisk.Path,
	)
	if err != nil {
		return Result{}, fmt.Errorf("find free GPT extent start: %w", err)
	}

	freeEnd, err := readUint(
		ctx,
		runner,
		"sudo",
		"sgdisk",
		"-E",
		input.Plan.TargetDisk.Path,
	)
	if err != nil {
		return Result{}, fmt.Errorf("find free GPT extent end: %w", err)
	}

	var availableSectors uint64
	if freeEnd >= freeStart {
		availableSectors = freeEnd - freeStart + 1
	}

	availableBytes := availableSectors * sectorSize
	requiredBytes := recovery.SizeBytes

	if availableBytes < requiredBytes {
		message := fmt.Sprintf(
			"Recovery needs %d bytes but the largest suitable unallocated GPT extent has %d bytes. "+
				"The existing OS layout has been left unchanged. "+
				"An offline storage resize is required before recovery provisioning can continue.",
			requiredBytes,
			availableBytes,
		)

		fmt.Fprintf(input.Out, "ACTION REQUIRED: %s\n", message)

		return Result{
			Status:         StatusOfflineResizeRequired,
			RequiredBytes:  requiredBytes,
			AvailableBytes: availableBytes,
			Message:        message,
		}, nil
	}

	fmt.Fprintf(
		input.Out,
		"PLAN: preserve existing GPT and partitions; create %s (%d bytes) in unallocated space.\n",
		recovery.Label,
		recovery.SizeBytes,
	)

	phrase := "CREATE-JODS-RECOVERY"
	if input.Unattended {
		fmt.Fprintln(
			input.Out,
			"UNATTENDED: explicit unattendedInstall=true authorizes recovery GPT creation.",
		)
	} else {
		if err := input.UI.Exact(
			ctx,
			fmt.Sprintf("Type %q to create the recovery partition", phrase),
			phrase,
		); err != nil {
			return Result{}, fmt.Errorf(
				"recovery partition creation not authorized: %w",
				err,
			)
		}
	}

	requiredSectors := requiredBytes / sectorSize
	number := strconv.FormatUint(uint64(recovery.Number), 10)

	if err := privilegedRun(
		ctx,
		runner,
		"sgdisk",
		"--new="+number+":"+strconv.FormatUint(freeStart, 10)+
			":+"+strconv.FormatUint(requiredSectors, 10),
		"--typecode="+number+":"+strings.ToLower(recovery.TypeGUID),
		"--change-name="+number+":"+recovery.Label,
		"--partition-guid="+number+":"+strings.ToLower(recovery.PARTUUID),
		input.Plan.TargetDisk.Path,
	); err != nil {
		return Result{}, fmt.Errorf("create canonical recovery GPT entry: %w", err)
	}

	if err := privilegedRun(
		ctx,
		runner,
		"blockdev",
		"--rereadpt",
		input.Plan.TargetDisk.Path,
	); err != nil {
		return Result{}, fmt.Errorf("reread partition table: %w", err)
	}

	if err := privilegedRun(
		ctx,
		runner,
		"udevadm",
		"settle",
	); err != nil {
		return Result{}, fmt.Errorf("wait for recovery partition device: %w", err)
	}

	after, err := inspectDisk(ctx, runner, input.Plan.TargetDisk.Path)
	if err != nil {
		return Result{}, fmt.Errorf("verify recovery GPT state: %w", err)
	}

	if err := verifyPreserved(before, after, recovery.Number); err != nil {
		return Result{}, err
	}

	actual, found := recoveryPartition(after, recovery)
	if !found {
		return Result{}, fmt.Errorf(
			"canonical recovery GPT entry was not verified after creation",
		)
	}

	afterGUID, err := readDiskGUID(
		ctx,
		runner,
		input.Plan.TargetDisk.Path,
	)
	if err != nil {
		return Result{}, err
	}

	if !strings.EqualFold(afterGUID, diskGUID) {
		return Result{}, fmt.Errorf(
			"GPT disk GUID changed while creating recovery partition",
		)
	}

	fmt.Fprintf(
		input.Out,
		"PASS: canonical recovery GPT entry created at %s; existing partitions preserved\n",
		actual.Path,
	)

	return Result{
		Status:         StatusCreated,
		PartitionPath:  actual.Path,
		RequiredBytes:  requiredBytes,
		AvailableBytes: availableBytes,
		Message:        "Canonical recovery partition created.",
	}, nil
}

func privilegedRun(
	ctx context.Context,
	runner commandRunner,
	command string,
	args ...string,
) error {
	return runner.Run(
		ctx,
		"sudo",
		append([]string{command}, args...)...,
	)
}

func readUint(
	ctx context.Context,
	runner commandRunner,
	name string,
	args ...string,
) (uint64, error) {
	raw, err := runner.Output(ctx, name, args...)
	if err != nil {
		return 0, err
	}

	value := strings.TrimSpace(string(raw))
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid numeric output %q", value)
	}

	return parsed, nil
}

func readDiskGUID(
	ctx context.Context,
	runner commandRunner,
	disk string,
) (string, error) {
	raw, err := runner.Output(
		ctx,
		"sudo",
		"sgdisk",
		"-p",
		disk,
	)
	if err != nil {
		return "", fmt.Errorf("read GPT disk GUID: %w", err)
	}

	match := diskGUIDPattern.FindSubmatch(raw)
	if len(match) != 2 {
		return "", fmt.Errorf("sgdisk did not report a GPT disk GUID")
	}

	return strings.ToLower(string(match[1])), nil
}

type lsblkTree struct {
	BlockDevices []lsblkDevice `json:"blockdevices"`
}

type lsblkDevice struct {
	Path        string        `json:"path"`
	Type        string        `json:"type"`
	Size        uint64        `json:"size"`
	Start       uint64        `json:"start"`
	Model       string        `json:"model"`
	Serial      string        `json:"serial"`
	WWN         string        `json:"wwn"`
	PTType      string        `json:"pttype"`
	PartN       uint          `json:"partn"`
	PartLabel   string        `json:"partlabel"`
	PartType    string        `json:"parttype"`
	PartUUID    string        `json:"partuuid"`
	MountPoints []string      `json:"mountpoints"`
	Children    []lsblkDevice `json:"children"`
}

type diskSnapshot struct {
	Disk       lsblkDevice
	Partitions map[uint]lsblkDevice
}

func inspectDisk(
	ctx context.Context,
	runner commandRunner,
	disk string,
) (diskSnapshot, error) {
	raw, err := runner.Output(
		ctx,
		"lsblk",
		"-J",
		"-b",
		"-p",
		"-o",
		"PATH,TYPE,SIZE,START,MODEL,SERIAL,WWN,PTTYPE,PARTN,PARTLABEL,PARTTYPE,PARTUUID,MOUNTPOINTS",
		"--",
		disk,
	)
	if err != nil {
		return diskSnapshot{}, fmt.Errorf("inspect GPT target: %w", err)
	}

	var tree lsblkTree
	if err := json.NewDecoder(bytes.NewReader(raw)).Decode(&tree); err != nil {
		return diskSnapshot{}, fmt.Errorf("decode GPT target metadata: %w", err)
	}

	if len(tree.BlockDevices) != 1 {
		return diskSnapshot{}, fmt.Errorf(
			"target resolved to %d top-level block devices",
			len(tree.BlockDevices),
		)
	}

	d := tree.BlockDevices[0]
	if strings.TrimSpace(d.Type) != "disk" {
		return diskSnapshot{}, fmt.Errorf("target %s is not a whole disk", disk)
	}

	snapshot := diskSnapshot{
		Disk:       d,
		Partitions: map[uint]lsblkDevice{},
	}

	for _, child := range d.Children {
		if strings.TrimSpace(child.Type) == "part" {
			snapshot.Partitions[child.PartN] = child
		}
	}

	return snapshot, nil
}

func validateTargetIdentity(
	expected diskplan.Disk,
	actual lsblkDevice,
) error {
	if strings.TrimSpace(expected.Path) != strings.TrimSpace(actual.Path) {
		return fmt.Errorf("target disk path changed")
	}

	if strings.TrimSpace(expected.Model) != strings.TrimSpace(actual.Model) {
		return fmt.Errorf("target disk model changed")
	}

	if expected.SizeBytes != actual.Size {
		return fmt.Errorf("target disk size changed")
	}

	if strings.TrimSpace(expected.WWN) != "" {
		if strings.TrimSpace(expected.WWN) != strings.TrimSpace(actual.WWN) {
			return fmt.Errorf("target disk WWN changed")
		}
		return nil
	}

	if strings.TrimSpace(expected.Serial) != strings.TrimSpace(actual.Serial) {
		return fmt.Errorf("target disk serial changed")
	}

	return nil
}

func recoveryPartition(
	snapshot diskSnapshot,
	expected diskplan.Partition,
) (lsblkDevice, bool) {
	for _, part := range snapshot.Partitions {
		if part.PartN != expected.Number {
			continue
		}

		if strings.TrimSpace(part.PartLabel) != expected.Label ||
			!strings.EqualFold(part.PartType, expected.TypeGUID) ||
			!strings.EqualFold(part.PartUUID, expected.PARTUUID) ||
			part.Size != expected.SizeBytes {
			return lsblkDevice{}, false
		}

		return part, true
	}

	return lsblkDevice{}, false
}

func ensureRecoverySlotFree(
	snapshot diskSnapshot,
	number uint,
) error {
	if part, found := snapshot.Partitions[number]; found {
		return fmt.Errorf(
			"canonical recovery partition number %d is already occupied by %s (%s)",
			number,
			part.Path,
			part.PartLabel,
		)
	}
	return nil
}

func verifyPreserved(
	before diskSnapshot,
	after diskSnapshot,
	recoveryNumber uint,
) error {
	for number, expected := range before.Partitions {
		if number == recoveryNumber {
			continue
		}

		actual, found := after.Partitions[number]
		if !found {
			return fmt.Errorf(
				"existing partition %d disappeared during recovery provisioning",
				number,
			)
		}

		if expected.Path != actual.Path ||
			expected.Size != actual.Size ||
			expected.Start != actual.Start ||
			expected.PartLabel != actual.PartLabel ||
			!strings.EqualFold(expected.PartType, actual.PartType) ||
			!strings.EqualFold(expected.PartUUID, actual.PartUUID) {
			return fmt.Errorf(
				"existing partition %d changed during recovery provisioning",
				number,
			)
		}
	}

	return nil
}
