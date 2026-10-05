package recoveryprovision

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/bakanura/gjallarOS/internal/installer/diskplan"
)

type discoveryRunner struct{}

func (discoveryRunner) Output(
	_ context.Context,
	name string,
	_ ...string,
) ([]byte, error) {
	switch name {
	case "findmnt":
		return []byte("/dev/mapper/cryptroot btrfs rw,relatime\n"), nil
	case "cryptsetup":
		return []byte("  type:    LUKS2\n  device:  /dev/vda2\n"), nil
	}
	return nil, errors.New("stop after backing-device check")
}

func (discoveryRunner) Run(context.Context, string, ...string) error {
	return errors.New("unexpected Run")
}

func (discoveryRunner) RunInput(
	context.Context,
	[]byte,
	string,
	...string,
) error {
	return errors.New("unexpected RunInput")
}

func TestExecuteMaintenanceResolvesPARTUUIDLink(t *testing.T) {
	previous := evalSymlinks
	t.Cleanup(func() { evalSymlinks = previous })

	var resolved string
	evalSymlinks = func(path string) (string, error) {
		resolved = path
		return "/dev/vda2", nil
	}

	plan := diskplan.Plan{
		TargetDisk: diskplan.Disk{Path: "/dev/vda"},
		Recovery:   &diskplan.Partition{SizeBytes: RecoveryBytes},
	}
	plan.Root.Partition.PARTUUID = "9f207f86-3d1d-45b4-9bba-1596a8b2be56"
	plan.Root.Encryption.MappingName = "cryptroot"

	_, err := ExecuteMaintenance(
		context.Background(),
		discoveryRunner{},
		MaintenanceInput{Plan: plan, Passphrase: []byte("x")},
	)

	if resolved != "/dev/disk/by-partuuid/9f207f86-3d1d-45b4-9bba-1596a8b2be56" {
		t.Fatalf("resolved %q", resolved)
	}
	if err == nil || strings.Contains(err.Error(), "backing-device mismatch") {
		t.Fatalf("err = %v", err)
	}
}
