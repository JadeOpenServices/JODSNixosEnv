// Package recoveryresize plans the Btrfs/LUKS2/GPT boundary transition
// needed to create unallocated space for the canonical GJAL-31 recovery
// partition.
//
// This package is deliberately planning-only. It contains no command capable
// of resizing a filesystem, dm-crypt mapping, partition table, or recovery
// partition. Execution belongs to a later, separately gated stage.
package recoveryresize

import (
	"errors"
	"fmt"
	"strings"
)

const (
	StageCheckingCapability    = "checking-recovery-capability"
	StageCheckingSpace         = "checking-recovery-space"
	StagePreparingResize       = "preparing-recovery-resize"
	StageValidatingEnvironment = "validating-resize-environment"
	StageResizingRootStorage   = "resizing-root-storage"
	StageCreatingRecovery      = "creating-recovery-partition"
	StageVerifyingRecovery     = "verifying-recovery-layout"
	StageRecoveryReady         = "recovery-partition-ready"

	FilesystemBtrfs = "btrfs"
)

var ErrUnsupportedFilesystem = errors.New(
	"recovery partitioning requires a Btrfs root filesystem",
)

type Topology struct {
	DiskPath           string
	DiskGUID           string
	DiskSizeBytes      uint64
	LogicalSectorBytes uint64

	RootPartitionPath     string
	RootPartitionNumber   uint64
	RootPartitionTypeGUID string
	RootPARTUUID          string
	RootStartBytes        uint64
	RootEndBytes          uint64
	RootSizeBytes         uint64

	MappingName string
	MappingPath string
	LUKSUUID    string

	// LUKSPayloadOffsetBytes is the byte offset at which encrypted payload
	// data begins inside the LUKS partition. It is preserved by resizing.
	LUKSPayloadOffsetBytes uint64

	RootFilesystem string
	RootMountpoint string
	RootMountedRW  bool

	BtrfsUUID               string
	BtrfsDeviceID           uint64
	BtrfsDeviceBytes        uint64
	BtrfsUsedBytes          uint64
	BtrfsExclusiveOperation string
	SwapActive              bool
}

type Requirements struct {
	RecoveryBytes uint64

	// SafetyMarginBytes is intentionally left unallocated between the new
	// root end and recovery partition. This makes the storage contraction
	// smaller than the absolute mathematical maximum.
	SafetyMarginBytes uint64

	// FilesystemHeadroomBytes must remain free inside Btrfs after shrink.
	// This is separate from the outer GPT safety margin.
	FilesystemHeadroomBytes uint64

	// AlignmentBytes is the required partition-end alignment.
	AlignmentBytes uint64
}

type Plan struct {
	Stage string

	OriginalRootStartBytes uint64
	OriginalRootEndBytes   uint64
	OriginalRootSizeBytes  uint64

	NewRootStartBytes uint64
	NewRootEndBytes   uint64
	NewRootSizeBytes  uint64

	ShrinkBytes uint64

	TargetCryptPayloadBytes uint64
	TargetBtrfsDeviceBytes  uint64

	RecoveryBytes     uint64
	SafetyMarginBytes uint64
	AlignmentBytes    uint64

	ExpectedDiskGUID  string
	ExpectedPARTUUID  string
	ExpectedLUKSUUID  string
	ExpectedBtrfsUUID string
}

func BuildPlan(topology Topology, req Requirements) (Plan, error) {
	if err := ValidateTopology(topology); err != nil {
		return Plan{}, err
	}
	if err := validateRequirements(req); err != nil {
		return Plan{}, err
	}

	shrinkBytes, err := add(req.RecoveryBytes, req.SafetyMarginBytes)
	if err != nil {
		return Plan{}, fmt.Errorf("calculate recovery shrink: %w", err)
	}
	shrinkBytes = alignUp(shrinkBytes, req.AlignmentBytes)

	if shrinkBytes >= topology.RootSizeBytes {
		return Plan{}, fmt.Errorf(
			"requested recovery shrink %d is not smaller than root partition %d",
			shrinkBytes,
			topology.RootSizeBytes,
		)
	}

	newRootSize := topology.RootSizeBytes - shrinkBytes
	if newRootSize <= topology.LUKSPayloadOffsetBytes {
		return Plan{}, fmt.Errorf(
			"new root size %d does not preserve LUKS payload offset %d",
			newRootSize,
			topology.LUKSPayloadOffsetBytes,
		)
	}

	targetPayload := newRootSize - topology.LUKSPayloadOffsetBytes

	if targetPayload > topology.BtrfsDeviceBytes {
		return Plan{}, fmt.Errorf(
			"planned encrypted payload %d exceeds current Btrfs device size %d",
			targetPayload,
			topology.BtrfsDeviceBytes,
		)
	}

	minimumBtrfs, err := add(
		topology.BtrfsUsedBytes,
		req.FilesystemHeadroomBytes,
	)
	if err != nil {
		return Plan{}, fmt.Errorf(
			"calculate minimum Btrfs size: %w",
			err,
		)
	}
	if targetPayload < minimumBtrfs {
		return Plan{}, fmt.Errorf(
			"insufficient Btrfs headroom: target=%d used=%d required_headroom=%d",
			targetPayload,
			topology.BtrfsUsedBytes,
			req.FilesystemHeadroomBytes,
		)
	}

	newEnd := topology.RootStartBytes + newRootSize
	if newEnd >= topology.RootEndBytes {
		return Plan{}, errors.New(
			"planned root end was not reduced",
		)
	}

	if newEnd%req.AlignmentBytes != 0 {
		return Plan{}, fmt.Errorf(
			"planned root end %d is not aligned to %d bytes",
			newEnd,
			req.AlignmentBytes,
		)
	}

	return Plan{
		Stage: StagePreparingResize,

		OriginalRootStartBytes: topology.RootStartBytes,
		OriginalRootEndBytes:   topology.RootEndBytes,
		OriginalRootSizeBytes:  topology.RootSizeBytes,

		NewRootStartBytes: topology.RootStartBytes,
		NewRootEndBytes:   newEnd,
		NewRootSizeBytes:  newRootSize,

		ShrinkBytes: shrinkBytes,

		TargetCryptPayloadBytes: targetPayload,
		TargetBtrfsDeviceBytes:  targetPayload,

		RecoveryBytes:     req.RecoveryBytes,
		SafetyMarginBytes: req.SafetyMarginBytes,
		AlignmentBytes:    req.AlignmentBytes,

		ExpectedDiskGUID:  topology.DiskGUID,
		ExpectedPARTUUID:  topology.RootPARTUUID,
		ExpectedLUKSUUID:  topology.LUKSUUID,
		ExpectedBtrfsUUID: topology.BtrfsUUID,
	}, nil
}

func ValidateTopology(t Topology) error {
	fs := strings.ToLower(strings.TrimSpace(t.RootFilesystem))
	if fs != FilesystemBtrfs {
		return fmt.Errorf(
			"%w. Current filesystem: %s. No disk changes were made.",
			ErrUnsupportedFilesystem,
			printableFilesystem(fs),
		)
	}

	requiredStrings := map[string]string{
		"disk path":           t.DiskPath,
		"disk GUID":           t.DiskGUID,
		"root partition path": t.RootPartitionPath,
		"root PARTUUID":       t.RootPARTUUID,
		"mapping name":        t.MappingName,
		"mapping path":        t.MappingPath,
		"LUKS UUID":           t.LUKSUUID,
		"Btrfs UUID":          t.BtrfsUUID,
		"root mountpoint":     t.RootMountpoint,
	}
	for name, value := range requiredStrings {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", name)
		}
	}

	if t.DiskSizeBytes == 0 {
		return errors.New("disk size is required")
	}
	if t.LogicalSectorBytes == 0 {
		return errors.New("logical sector size is required")
	}
	if t.RootPartitionNumber == 0 {
		return errors.New("root partition number is required")
	}
	if strings.TrimSpace(t.RootPartitionTypeGUID) == "" {
		return errors.New("root partition type GUID is required")
	}
	if t.RootStartBytes == 0 {
		return errors.New("root partition start is required")
	}
	if t.RootEndBytes <= t.RootStartBytes {
		return errors.New("root partition geometry is invalid")
	}
	if t.RootSizeBytes != t.RootEndBytes-t.RootStartBytes {
		return fmt.Errorf(
			"root partition size mismatch: geometry=%d reported=%d",
			t.RootEndBytes-t.RootStartBytes,
			t.RootSizeBytes,
		)
	}
	if t.RootEndBytes > t.DiskSizeBytes {
		return errors.New("root partition extends past disk end")
	}
	if t.LUKSPayloadOffsetBytes == 0 {
		return errors.New("LUKS payload offset is required")
	}
	if t.LUKSPayloadOffsetBytes >= t.RootSizeBytes {
		return errors.New("LUKS payload offset exceeds root partition")
	}
	if t.BtrfsDeviceID == 0 {
		return errors.New("Btrfs device ID is required")
	}
	if t.BtrfsDeviceBytes == 0 {
		return errors.New("Btrfs device size is required")
	}
	if t.BtrfsUsedBytes >= t.BtrfsDeviceBytes {
		return errors.New(
			"Btrfs used space leaves no shrinkable capacity",
		)
	}
	if !t.RootMountedRW {
		return errors.New(
			"Btrfs root must be mounted read-write for live filesystem shrink planning",
		)
	}

	exclusive := strings.ToLower(
		strings.TrimSpace(t.BtrfsExclusiveOperation),
	)
	switch exclusive {
	case "", "none":
	default:
		return fmt.Errorf(
			"Btrfs exclusive operation %q is active",
			t.BtrfsExclusiveOperation,
		)
	}

	if t.SwapActive {
		return errors.New(
			"active swap is not permitted during recovery resize planning",
		)
	}

	return nil
}

func validateRequirements(r Requirements) error {
	if r.RecoveryBytes == 0 {
		return errors.New("recovery size is required")
	}
	if r.SafetyMarginBytes == 0 {
		return errors.New("explicit GPT safety margin is required")
	}
	if r.FilesystemHeadroomBytes == 0 {
		return errors.New(
			"explicit Btrfs filesystem headroom is required",
		)
	}
	if r.AlignmentBytes == 0 {
		return errors.New("partition alignment is required")
	}
	return nil
}

func printableFilesystem(fs string) string {
	if fs == "" {
		return "unknown"
	}
	return fs
}

func alignUp(value, alignment uint64) uint64 {
	if alignment == 0 {
		return value
	}
	rem := value % alignment
	if rem == 0 {
		return value
	}
	return value + alignment - rem
}

func add(a, b uint64) (uint64, error) {
	sum := a + b
	if sum < a {
		return 0, errors.New("size arithmetic overflow")
	}
	return sum, nil
}
