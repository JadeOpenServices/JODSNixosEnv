package recoveryresize

import (
	"errors"
	"strings"
	"testing"
)

const GiB = uint64(1024 * 1024 * 1024)
const MiB = uint64(1024 * 1024)

func topology() Topology {
	start := GiB
	size := 900 * GiB

	return Topology{
		DiskPath:           "/dev/nvme0n1",
		DiskGUID:           "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		DiskSizeBytes:      1000 * GiB,
		LogicalSectorBytes: 512,

		RootPartitionPath:     "/dev/nvme0n1p2",
		RootPartitionNumber:   2,
		RootPartitionTypeGUID: "ca7d7ccb-63ed-4c53-861c-1742536059cc",
		RootPARTUUID:          "11111111-2222-3333-4444-555555555555",
		RootStartBytes:        start,
		RootEndBytes:          start + size,
		RootSizeBytes:         size,

		MappingName: "cryptroot",
		MappingPath: "/dev/mapper/cryptroot",
		LUKSUUID:    "22222222-3333-4444-5555-666666666666",

		LUKSPayloadOffsetBytes: 16 * MiB,

		RootFilesystem: "btrfs",
		RootMountpoint: "/mnt",
		RootMountedRW:  true,

		BtrfsUUID:               "33333333-4444-5555-6666-777777777777",
		BtrfsDeviceID:           1,
		BtrfsDeviceBytes:        size - 16*MiB,
		BtrfsUsedBytes:          400 * GiB,
		BtrfsExclusiveOperation: "none",
		SwapActive:              false,
	}
}

func requirements() Requirements {
	return Requirements{
		RecoveryBytes:           4 * GiB,
		SafetyMarginBytes:       1 * GiB,
		FilesystemHeadroomBytes: 8 * GiB,
		AlignmentBytes:          MiB,
	}
}

func TestBuildPlanPreservesRootStartAndIdentity(t *testing.T) {
	top := topology()
	plan, err := BuildPlan(top, requirements())
	if err != nil {
		t.Fatal(err)
	}

	if plan.NewRootStartBytes != top.RootStartBytes {
		t.Fatalf(
			"root start moved: got=%d want=%d",
			plan.NewRootStartBytes,
			top.RootStartBytes,
		)
	}
	if plan.NewRootEndBytes >= top.RootEndBytes {
		t.Fatal("root end was not reduced")
	}
	if plan.ExpectedDiskGUID != top.DiskGUID {
		t.Fatal("disk GUID not preserved")
	}
	if plan.ExpectedPARTUUID != top.RootPARTUUID {
		t.Fatal("PARTUUID not preserved")
	}
	if plan.ExpectedLUKSUUID != top.LUKSUUID {
		t.Fatal("LUKS UUID not preserved")
	}
	if plan.ExpectedBtrfsUUID != top.BtrfsUUID {
		t.Fatal("Btrfs UUID not preserved")
	}
}

func TestBuildPlanCreatesOnlyRequestedSpacePlusMargin(t *testing.T) {
	top := topology()
	req := requirements()

	plan, err := BuildPlan(top, req)
	if err != nil {
		t.Fatal(err)
	}

	want := req.RecoveryBytes + req.SafetyMarginBytes
	if plan.ShrinkBytes != want {
		t.Fatalf(
			"shrink=%d want=%d",
			plan.ShrinkBytes,
			want,
		)
	}
}

func TestNonBtrfsFailsClosed(t *testing.T) {
	top := topology()
	top.RootFilesystem = "ext4"

	_, err := BuildPlan(top, requirements())
	if !errors.Is(err, ErrUnsupportedFilesystem) {
		t.Fatalf("error=%v", err)
	}
	if !strings.Contains(
		err.Error(),
		"Current filesystem: ext4. No disk changes were made.",
	) {
		t.Fatalf("unexpected UX: %v", err)
	}
}

func TestInsufficientBtrfsHeadroomFails(t *testing.T) {
	top := topology()
	top.BtrfsUsedBytes = 898 * GiB

	_, err := BuildPlan(top, requirements())
	if err == nil ||
		!strings.Contains(err.Error(), "insufficient Btrfs headroom") {
		t.Fatalf("error=%v", err)
	}
}

func TestActiveSwapFailsClosed(t *testing.T) {
	top := topology()
	top.SwapActive = true

	_, err := BuildPlan(top, requirements())
	if err == nil || !strings.Contains(err.Error(), "active swap") {
		t.Fatalf("error=%v", err)
	}
}

func TestConcurrentBtrfsOperationFailsClosed(t *testing.T) {
	top := topology()
	top.BtrfsExclusiveOperation = "balance"

	_, err := BuildPlan(top, requirements())
	if err == nil ||
		!strings.Contains(err.Error(), "exclusive operation") {
		t.Fatalf("error=%v", err)
	}
}

func TestGeometryMismatchFailsClosed(t *testing.T) {
	top := topology()
	top.RootSizeBytes++

	_, err := BuildPlan(top, requirements())
	if err == nil ||
		!strings.Contains(err.Error(), "size mismatch") {
		t.Fatalf("error=%v", err)
	}
}

func TestLUKSPayloadOffsetPreserved(t *testing.T) {
	top := topology()

	plan, err := BuildPlan(top, requirements())
	if err != nil {
		t.Fatal(err)
	}

	want := plan.NewRootSizeBytes -
		top.LUKSPayloadOffsetBytes

	if plan.TargetCryptPayloadBytes != want {
		t.Fatalf(
			"payload=%d want=%d",
			plan.TargetCryptPayloadBytes,
			want,
		)
	}
	if plan.TargetBtrfsDeviceBytes != want {
		t.Fatalf(
			"btrfs target=%d want=%d",
			plan.TargetBtrfsDeviceBytes,
			want,
		)
	}
}
