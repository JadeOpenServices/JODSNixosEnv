package freshgpt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"

	"github.com/bakanura/gjallarOS/internal/installer/diskplan"
)

type Input struct {
	Plan diskplan.Plan
	Out  io.Writer
}

type Result struct {
	DiskPath         string
	ESPPARTUUID      string
	RootPARTUUID     string
	RecoveryPARTUUID string
}

type commandRunner interface {
	Run(context.Context, string, ...string) error
	Output(context.Context, string, ...string) ([]byte, error)
}

type execRunner struct{}

func (execRunner) Run(
	ctx context.Context,
	name string,
	args ...string,
) error {
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
		return Result{}, fmt.Errorf("fresh GPT output writer is required")
	}
	if err := input.Plan.Validate(); err != nil {
		return Result{}, fmt.Errorf(
			"validate canonical disk plan: %w",
			err,
		)
	}

	p := input.Plan
	disk := strings.TrimSpace(p.TargetDisk.Path)

	fmt.Fprintf(
		input.Out,
		"STAGE: provisioning fresh canonical GPT on %s\n",
		disk,
	)

	// Destructive authority belongs to installconfirm. This package has no
	// prompt/bypass of its own and must only be called after that gate.
	//
	// sgdisk --zap-all only removes partition tables. A previously used disk
	// (an earlier attempt, or any OS with an ESP at 1 MiB) keeps filesystem
	// and LUKS signatures at the offsets the canonical layout reuses, which
	// verify() then correctly refuses. Erase them while the old partitions
	// still exist; verify() keeps rejecting anything that survives.
	if err := eraseExistingSignatures(ctx, runner, disk); err != nil {
		return Result{}, err
	}

	if err := runner.Run(
		ctx,
		"sudo",
		"sgdisk",
		"--zap-all",
		disk,
	); err != nil {
		return Result{}, fmt.Errorf("erase target GPT: %w", err)
	}

	if err := runner.Run(
		ctx,
		"sudo",
		"sgdisk",
		"--disk-guid="+strings.ToLower(p.TargetDisk.GPTDiskGUID),
		disk,
	); err != nil {
		return Result{}, fmt.Errorf("set canonical GPT disk GUID: %w", err)
	}

	// Allocate in canonical order. sgdisk's aligned default start is used for
	// partition 1; subsequent partitions begin at the next aligned free sector.
	if err := createPartition(ctx, runner, disk, p.ESP); err != nil {
		return Result{}, fmt.Errorf("create ESP: %w", err)
	}
	if err := createPartition(
		ctx,
		runner,
		disk,
		p.Root.Partition,
	); err != nil {
		return Result{}, fmt.Errorf("create root partition: %w", err)
	}
	if p.Recovery != nil {
		if err := createPartition(
			ctx,
			runner,
			disk,
			*p.Recovery,
		); err != nil {
			return Result{}, fmt.Errorf(
				"create recovery partition: %w",
				err,
			)
		}
	}

	if err := runner.Run(
		ctx,
		"sudo",
		"partprobe",
		disk,
	); err != nil {
		return Result{}, fmt.Errorf("reread fresh GPT: %w", err)
	}
	if err := runner.Run(
		ctx,
		"sudo",
		"udevadm",
		"settle",
	); err != nil {
		return Result{}, fmt.Errorf("settle fresh GPT devices: %w", err)
	}

	if err := verify(ctx, runner, p); err != nil {
		return Result{}, err
	}

	result := Result{
		DiskPath:     disk,
		ESPPARTUUID:  p.ESP.PARTUUID,
		RootPARTUUID: p.Root.Partition.PARTUUID,
	}
	if p.Recovery != nil {
		result.RecoveryPARTUUID = p.Recovery.PARTUUID
	}

	fmt.Fprintln(
		input.Out,
		"PASS: canonical fresh GPT created and verified",
	)

	return result, nil
}

func eraseExistingSignatures(
	ctx context.Context,
	runner commandRunner,
	disk string,
) error {
	raw, err := runner.Output(
		ctx,
		"lsblk", "-J", "-p", "--tree", "-o", "PATH,TYPE", "--", disk,
	)
	if err != nil {
		return fmt.Errorf("inventory existing target partitions: %w", err)
	}
	var existing tree
	if err := json.NewDecoder(bytes.NewReader(raw)).Decode(&existing); err != nil {
		return fmt.Errorf("decode existing target partitions: %w", err)
	}
	if len(existing.BlockDevices) != 1 ||
		strings.TrimSpace(existing.BlockDevices[0].Path) != disk {
		return fmt.Errorf("existing partition inventory did not resolve %s", disk)
	}

	for _, part := range existing.BlockDevices[0].Children {
		if part.Type != "part" {
			continue
		}
		if !strings.HasPrefix(part.Path, "/dev/") {
			return fmt.Errorf("unexpected existing partition path %q", part.Path)
		}
		if err := runner.Run(ctx, "sudo", "wipefs", "--all", "--", part.Path); err != nil {
			return fmt.Errorf("erase signatures on %s: %w", part.Path, err)
		}
	}
	if err := runner.Run(ctx, "sudo", "wipefs", "--all", "--", disk); err != nil {
		return fmt.Errorf("erase signatures on %s: %w", disk, err)
	}
	return nil
}

func createPartition(
	ctx context.Context,
	runner commandRunner,
	disk string,
	part diskplan.Partition,
) error {
	if part.SizeBytes%(1024*1024) != 0 {
		return fmt.Errorf(
			"partition %d size %d is not MiB aligned",
			part.Number,
			part.SizeBytes,
		)
	}

	number := strconv.FormatUint(uint64(part.Number), 10)
	sizeMiB := part.SizeBytes / (1024 * 1024)

	args := []string{
		"sgdisk",
		"--new=" + number + ":0:+" + strconv.FormatUint(sizeMiB, 10) + "M",
		"--typecode=" + number + ":" + strings.ToLower(part.TypeGUID),
		"--change-name=" + number + ":" + part.Label,
		"--partition-guid=" + number + ":" + strings.ToLower(part.PARTUUID),
		disk,
	}

	return runner.Run(
		ctx,
		"sudo",
		args...,
	)
}

type tree struct {
	BlockDevices []device `json:"blockdevices"`
}

type device struct {
	Path     string   `json:"path"`
	Type     string   `json:"type"`
	PTType   string   `json:"pttype"`
	PTUUID   string   `json:"ptuuid"`
	Children []device `json:"children"`

	PartN     uint   `json:"partn"`
	PartLabel string `json:"partlabel"`
	PartType  string `json:"parttype"`
	PartUUID  string `json:"partuuid"`
	Size      uint64 `json:"size"`
	FSType    string `json:"fstype"`
}

func verify(
	ctx context.Context,
	runner commandRunner,
	plan diskplan.Plan,
) error {
	raw, err := runner.Output(
		ctx,
		"lsblk",
		"-J",
		"-b",
		"-p",
		"-o",
		"PATH,TYPE,SIZE,FSTYPE,PTTYPE,PTUUID,PARTN,PARTLABEL,PARTTYPE,PARTUUID",
		"--",
		plan.TargetDisk.Path,
	)
	if err != nil {
		return fmt.Errorf("verify fresh GPT with lsblk: %w", err)
	}

	var got tree
	if err := json.NewDecoder(bytes.NewReader(raw)).Decode(&got); err != nil {
		return fmt.Errorf("decode fresh GPT verification: %w", err)
	}
	var disk *device
	for i := range got.BlockDevices {
		candidate := &got.BlockDevices[i]
		if candidate.Type != "disk" {
			continue
		}
		if strings.TrimSpace(candidate.Path) != strings.TrimSpace(plan.TargetDisk.Path) {
			continue
		}

		if disk != nil {
			return fmt.Errorf(
				"fresh GPT verification resolved target disk %q more than once",
				plan.TargetDisk.Path,
			)
		}
		disk = candidate
	}

	if disk == nil {
		return fmt.Errorf(
			"fresh GPT verification did not resolve target disk %q",
			plan.TargetDisk.Path,
		)
	}

	if disk.Type != "disk" {
		return fmt.Errorf(
			"fresh GPT target changed block type to %q",
			disk.Type,
		)
	}
	if !strings.EqualFold(disk.PTType, "gpt") {
		return fmt.Errorf(
			"fresh target partition-table type is %q",
			disk.PTType,
		)
	}
	if !strings.EqualFold(
		disk.PTUUID,
		plan.TargetDisk.GPTDiskGUID,
	) {
		return fmt.Errorf(
			"fresh GPT disk GUID mismatch: got=%q expected=%q",
			disk.PTUUID,
			plan.TargetDisk.GPTDiskGUID,
		)
	}

	expected := map[uint]diskplan.Partition{
		plan.ESP.Number:            plan.ESP,
		plan.Root.Partition.Number: plan.Root.Partition,
	}
	if plan.Recovery != nil {
		expected[plan.Recovery.Number] = *plan.Recovery
	}

	actual := map[uint]device{}

	addPartition := func(part device) error {
		if part.Type != "part" {
			return nil
		}
		if part.PartN == 0 {
			return fmt.Errorf(
				"fresh GPT verification found partition without partition number",
			)
		}
		if _, exists := actual[part.PartN]; exists {
			return fmt.Errorf(
				"fresh GPT verification resolved partition %d more than once",
				part.PartN,
			)
		}
		actual[part.PartN] = part
		return nil
	}

	for _, child := range disk.Children {
		if err := addPartition(child); err != nil {
			return err
		}
	}

	for _, top := range got.BlockDevices {
		if top.Type != "part" {
			continue
		}
		if err := addPartition(top); err != nil {
			return err
		}
	}

	if len(actual) != len(expected) {
		return fmt.Errorf(
			"fresh GPT partition count mismatch: got=%d expected=%d",
			len(actual),
			len(expected),
		)
	}

	for number, want := range expected {
		gotPart, ok := actual[number]
		if !ok {
			return fmt.Errorf(
				"canonical partition %d is missing",
				number,
			)
		}
		if gotPart.Size != want.SizeBytes {
			return fmt.Errorf(
				"partition %d size mismatch: got=%d expected=%d",
				number,
				gotPart.Size,
				want.SizeBytes,
			)
		}
		if gotPart.PartLabel != want.Label {
			return fmt.Errorf(
				"partition %d label mismatch: got=%q expected=%q",
				number,
				gotPart.PartLabel,
				want.Label,
			)
		}
		if !strings.EqualFold(
			gotPart.PartType,
			want.TypeGUID,
		) {
			return fmt.Errorf(
				"partition %d type GUID mismatch",
				number,
			)
		}
		if !strings.EqualFold(
			gotPart.PartUUID,
			want.PARTUUID,
		) {
			return fmt.Errorf(
				"partition %d PARTUUID mismatch",
				number,
			)
		}
		if strings.TrimSpace(gotPart.FSType) != "" {
			return fmt.Errorf(
				"fresh partition %d unexpectedly contains filesystem %q",
				number,
				gotPart.FSType,
			)
		}
	}

	return nil
}
