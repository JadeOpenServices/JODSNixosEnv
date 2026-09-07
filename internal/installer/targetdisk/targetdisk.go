// Package targetdisk performs read-only validation of a candidate
// bare-metal installation target.
//
// It never partitions, formats, encrypts, mounts, wipes, or otherwise modifies
// the selected device. Its result describes observed physical identity only;
// the canonical intended installation layout remains owned by diskplan.
package targetdisk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

const nixOSISOLabel = "NIXOS_ISO"

var installationMediaMounts = map[string]bool{
	"/iso":               true,
	"/run/initramfs/iso": true,
	"/run/iso":           true,
}

// Input describes the candidate disk and optional minimum acceptable size.
//
// MinSizeBytes may be zero when the caller has no additional size policy.
// The physical device must still report a non-zero size.
type Input struct {
	Path         string
	MinSizeBytes uint64
}

// Result is the observed, non-secret identity of a validated physical disk.
//
// GPT identity is deliberately absent: GJAL-29 validates the existing physical
// target before the later canonical GPT provisioning stage creates/plans GPT
// identity.
type Result struct {
	Path      string
	Model     string
	Serial    string
	WWN       string
	SizeBytes uint64
}

type commandRunner interface {
	Output(context.Context, string, ...string) ([]byte, error)
}

type execRunner struct{}

func (execRunner) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}

// Validate inspects candidate using read-only lsblk metadata and fails closed
// when it cannot prove that the target is a suitable whole physical disk.
func Validate(ctx context.Context, input Input) (Result, error) {
	return validate(ctx, input, execRunner{})
}

func validate(ctx context.Context, input Input, runner commandRunner) (Result, error) {
	path := strings.TrimSpace(input.Path)
	if path == "" {
		return Result{}, fmt.Errorf("target disk path is required")
	}
	if !filepath.IsAbs(path) || filepath.Clean(path) == "/" ||
		!strings.HasPrefix(filepath.Clean(path), "/dev/") {
		return Result{}, fmt.Errorf(
			"target disk path must identify an absolute device under /dev: %q",
			input.Path,
		)
	}

	raw, err := runner.Output(
		ctx,
		"lsblk",
		"-J",
		"-b",
		"-p",
		"-o",
		"PATH,TYPE,MODEL,SERIAL,WWN,SIZE,LABEL,MOUNTPOINTS",
		"--",
		path,
	)
	if err != nil {
		return Result{}, fmt.Errorf("inspect target disk %q with lsblk: %w", path, err)
	}

	var tree lsblkTree
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := decoder.Decode(&tree); err != nil {
		return Result{}, fmt.Errorf("decode lsblk target metadata: %w", err)
	}
	if len(tree.BlockDevices) != 1 {
		return Result{}, fmt.Errorf(
			"target disk %q resolved to %d block devices; expected exactly one",
			path,
			len(tree.BlockDevices),
		)
	}

	disk := tree.BlockDevices[0]
	if strings.TrimSpace(disk.Type) != "disk" {
		return Result{}, fmt.Errorf(
			"target %q is not a whole block device (type %q)",
			path,
			disk.Type,
		)
	}
	if strings.TrimSpace(disk.Path) == "" {
		return Result{}, fmt.Errorf("lsblk did not report a canonical target disk path")
	}

	if containsRootMount(disk) {
		return Result{}, fmt.Errorf(
			"target disk %q backs the currently mounted root filesystem",
			disk.Path,
		)
	}
	if containsInstallationMedia(disk) {
		return Result{}, fmt.Errorf(
			"target disk %q contains the active installation media",
			disk.Path,
		)
	}

	model := strings.TrimSpace(disk.Model)
	serial := strings.TrimSpace(disk.Serial)
	wwn := strings.TrimSpace(disk.WWN)

	if model == "" {
		return Result{}, fmt.Errorf("target disk %q has no reported model", disk.Path)
	}
	if serial == "" && wwn == "" {
		return Result{}, fmt.Errorf(
			"target disk %q has no stable serial or WWN identity",
			disk.Path,
		)
	}
	if disk.Size == 0 {
		return Result{}, fmt.Errorf("target disk %q reports zero size", disk.Path)
	}
	if input.MinSizeBytes > 0 && disk.Size < input.MinSizeBytes {
		return Result{}, fmt.Errorf(
			"target disk %q is too small: %d bytes, require at least %d bytes",
			disk.Path,
			disk.Size,
			input.MinSizeBytes,
		)
	}

	return Result{
		Path:      filepath.Clean(disk.Path),
		Model:     model,
		Serial:    serial,
		WWN:       wwn,
		SizeBytes: disk.Size,
	}, nil
}

type lsblkTree struct {
	BlockDevices []blockDevice `json:"blockdevices"`
}

type blockDevice struct {
	Path        string        `json:"path"`
	Type        string        `json:"type"`
	Model       string        `json:"model"`
	Serial      string        `json:"serial"`
	WWN         string        `json:"wwn"`
	Size        uint64        `json:"size"`
	Label       string        `json:"label"`
	MountPoints []string      `json:"mountpoints"`
	Children    []blockDevice `json:"children"`
}

func containsRootMount(device blockDevice) bool {
	for _, mount := range device.MountPoints {
		if filepath.Clean(strings.TrimSpace(mount)) == "/" {
			return true
		}
	}
	for _, child := range device.Children {
		if containsRootMount(child) {
			return true
		}
	}
	return false
}

func containsInstallationMedia(device blockDevice) bool {
	if strings.EqualFold(strings.TrimSpace(device.Label), nixOSISOLabel) {
		return true
	}
	for _, mount := range device.MountPoints {
		mount = filepath.Clean(strings.TrimSpace(mount))
		if installationMediaMounts[mount] {
			return true
		}
	}
	for _, child := range device.Children {
		if containsInstallationMedia(child) {
			return true
		}
	}
	return false
}
