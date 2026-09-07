package recoveryresize

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
)

type executorRunner struct {
	outputs    map[string][]byte
	runs       [][]string
	inputRuns  [][]string
	inputSizes []int

	rootPartitionPath string
	rootMountpoint    string

	originalRootSize uint64
	newRootSize      uint64

	originalBtrfs []byte
	shrunkBtrfs   []byte

	btrfsShrunk     bool
	partitionShrunk bool
}

func (r *executorRunner) Output(
	_ context.Context,
	name string,
	args ...string,
) ([]byte, error) {
	call := append([]string{name}, args...)
	key := commandKey(name, args...)

	if key == commandKey(
		"btrfs",
		"filesystem",
		"show",
		"--raw",
		r.rootMountpoint,
	) {
		if r.btrfsShrunk {
			return r.shrunkBtrfs, nil
		}
		return r.originalBtrfs, nil
	}

	if key == commandKey(
		"blockdev",
		"--getsize64",
		r.rootPartitionPath,
	) {
		size := r.originalRootSize
		if r.partitionShrunk {
			size = r.newRootSize
		}
		return []byte(fmt.Sprintf("%d\n", size)), nil
	}

	out, ok := r.outputs[key]
	if !ok {
		// findmnt --mountpoint after the deliberate umount is expected to
		// report no mount. Model that as an empty successful output.
		if key == commandKey(
			"findmnt",
			"-nro",
			"SOURCE",
			"--mountpoint",
			"/mnt",
		) {
			return []byte(""), nil
		}

		return nil, fmt.Errorf(
			"unexpected executor output command: %v",
			call,
		)
	}

	return out, nil
}

func (r *executorRunner) Run(
	_ context.Context,
	name string,
	args ...string,
) error {
	call := append([]string{name}, args...)
	r.runs = append(r.runs, call)

	joined := strings.Join(call, " ")

	if strings.Contains(
		joined,
		"sudo btrfs filesystem resize",
	) {
		r.btrfsShrunk = true
	}

	if strings.Contains(
		joined,
		"sudo parted ",
	) && strings.Contains(
		joined,
		" resizepart ",
	) {
		r.partitionShrunk = true
	}

	return nil
}

func (r *executorRunner) RunInput(
	_ context.Context,
	input []byte,
	name string,
	args ...string,
) error {
	call := append([]string{name}, args...)

	r.inputRuns = append(
		r.inputRuns,
		call,
	)
	r.inputSizes = append(r.inputSizes, len(input))

	joined := strings.Join(call, " ")
	if strings.Contains(
		joined,
		"parted",
	) && strings.Contains(
		joined,
		" resizepart ",
	) {
		r.partitionShrunk = true
	}

	return nil
}

func executorFixture(
	t *testing.T,
) (Topology, Plan, *executorRunner) {
	t.Helper()

	top := topology()
	plan, err := BuildPlan(top, requirements())
	if err != nil {
		t.Fatal(err)
	}

	r := validDiscoveryRunner()

	// Discovery fixture needs new partition fields.
	r.outputs[commandKey(
		"lsblk",
		"-dnro",
		"PARTN",
		top.RootPartitionPath,
	)] = []byte("2\n")

	r.outputs[commandKey(
		"lsblk",
		"-dnro",
		"PARTTYPE",
		top.RootPartitionPath,
	)] = []byte(
		top.RootPartitionTypeGUID + "\n",
	)

	// Convert discovery fixture into a stateful executor fixture.
	//
	// Before any mutation:
	//   - Btrfs reports original size
	//   - partition reports original size
	//
	// After the relevant real command:
	//   - Btrfs reports the shrunk size
	//   - partition reports the shrunk size
	ex := &executorRunner{
		outputs:           map[string][]byte{},
		rootPartitionPath: top.RootPartitionPath,
		rootMountpoint:    top.RootMountpoint,
		originalRootSize:  top.RootSizeBytes,
		newRootSize:       plan.NewRootSizeBytes,
	}

	for key, value := range r.outputs {
		ex.outputs[key] = value
	}

	originalBtrfsKey := commandKey(
		"btrfs",
		"filesystem",
		"show",
		"--raw",
		top.RootMountpoint,
	)

	ex.originalBtrfs = append(
		[]byte(nil),
		ex.outputs[originalBtrfsKey]...,
	)

	if len(ex.originalBtrfs) == 0 {
		t.Fatal("original Btrfs discovery fixture is empty")
	}

	ex.shrunkBtrfs = []byte(fmt.Sprintf(
		`Label: 'GJALLAROS'  uuid: %s
	Total devices 1 FS bytes used %d
	devid    %d size %d used %d path %s
`,
		top.BtrfsUUID,
		top.BtrfsUsedBytes,
		top.BtrfsDeviceID,
		plan.TargetBtrfsDeviceBytes,
		top.BtrfsUsedBytes,
		top.MappingPath,
	))

	ex.outputs[commandKey(
		"lsblk",
		"-dnro",
		"PTUUID",
		top.DiskPath,
	)] = []byte(top.DiskGUID + "\n")

	ex.outputs[commandKey(
		"lsblk",
		"-dnro",
		"PARTUUID",
		top.RootPartitionPath,
	)] = []byte(top.RootPARTUUID + "\n")

	ex.outputs[commandKey(
		"lsblk",
		"-dnro",
		"PARTTYPE",
		top.RootPartitionPath,
	)] = []byte(top.RootPartitionTypeGUID + "\n")

	startSectors := top.RootStartBytes /
		top.LogicalSectorBytes
	ex.outputs[commandKey(
		"lsblk",
		"-dnro",
		"START",
		top.RootPartitionPath,
	)] = []byte(fmt.Sprintf("%d\n", startSectors))

	ex.outputs[commandKey(
		"sudo",
		"cryptsetup",
		"luksUUID",
		top.RootPartitionPath,
	)] = []byte(top.LUKSUUID + "\n")

	return top, plan, ex
}

func TestExecutorCommandOrder(t *testing.T) {
	top, plan, r := executorFixture(t)
	var out bytes.Buffer

	_, err := ExecuteFreshInstallerResize(
		context.Background(),
		r,
		ExecuteInput{
			Topology:   top,
			Plan:       plan,
			Passphrase: []byte("secret"),
			Out:        &out,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	joined := ""
	for _, call := range r.runs {
		joined += strings.Join(call, " ") + "\n"
	}

	requiredRunOrder := []string{
		"sudo btrfs filesystem resize",
		"sudo sync",
		"sudo umount /mnt",
		"sudo cryptsetup close cryptroot",
	}

	pos := -1
	for _, required := range requiredRunOrder {
		next := strings.Index(joined[pos+1:], required)
		if next < 0 {
			t.Fatalf(
				"missing ordered command %q:\n%s",
				required,
				joined,
			)
		}
		pos += next + 1
	}

	if len(r.inputRuns) != 2 {
		t.Fatalf(
			"expected parted confirmation + LUKS reopen stdin calls, got %d",
			len(r.inputRuns),
		)
	}

	parted := strings.Join(r.inputRuns[0], " ")
	if !strings.Contains(
		parted,
		"sudo env LC_ALL=C parted ---pretend-input-tty --align optimal",
	) || !strings.Contains(
		parted,
		" resizepart 2 ",
	) {
		t.Fatalf(
			"unexpected parted confirmation command: %s",
			parted,
		)
	}

	if r.inputSizes[0] != len("Yes\n") {
		t.Fatalf(
			"unexpected parted confirmation stdin size: %v",
			r.inputSizes,
		)
	}

	joinedAfterParted := ""
	for _, call := range r.runs {
		joinedAfterParted += strings.Join(call, " ") + "\n"
	}

	for _, required := range []string{
		"sudo partx --update --nr 2",
		"sudo udevadm settle",
		"sudo mount /dev/mapper/cryptroot /mnt",
	} {
		if !strings.Contains(joinedAfterParted, required) {
			t.Fatalf(
				"missing post-parted command %q:\n%s",
				required,
				joinedAfterParted,
			)
		}
	}

	// LUKS reopen is the second stdin operation.

	reopen := strings.Join(r.inputRuns[1], " ")
	if !strings.Contains(
		reopen,
		"sudo cryptsetup open --type luks2 --key-file -",
	) {
		t.Fatalf("unexpected reopen command: %s", reopen)
	}
}

func TestExecutorNeverPutsSecretInArguments(t *testing.T) {
	top, plan, r := executorFixture(t)

	secret := "DO-NOT-LOG-THIS"

	_, err := ExecuteFreshInstallerResize(
		context.Background(),
		r,
		ExecuteInput{
			Topology:   top,
			Plan:       plan,
			Passphrase: []byte(secret),
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	for _, calls := range [][][]string{
		r.runs,
		r.inputRuns,
	} {
		for _, call := range calls {
			if strings.Contains(
				strings.Join(call, " "),
				secret,
			) {
				t.Fatalf(
					"secret appeared in argv: %v",
					call,
				)
			}
		}
	}

	if len(r.inputSizes) != 2 ||
		r.inputSizes[0] != len("Yes\n") ||
		r.inputSizes[1] != len(secret) {
		t.Fatalf(
			"stdin accounting=%v",
			r.inputSizes,
		)
	}
}

func TestExecutorRejectsRunningRoot(t *testing.T) {
	top, plan, r := executorFixture(t)
	top.RootMountpoint = "/"

	_, err := ExecuteFreshInstallerResize(
		context.Background(),
		r,
		ExecuteInput{
			Topology:   top,
			Plan:       plan,
			Passphrase: []byte("secret"),
		},
	)
	if err == nil ||
		!strings.Contains(err.Error(), "fresh-install only") {
		t.Fatalf("error=%v", err)
	}

	if len(r.runs) != 0 {
		t.Fatalf("mutation commands ran: %v", r.runs)
	}
}

func TestExecutorRejectsMovedRootStart(t *testing.T) {
	top, plan, r := executorFixture(t)
	plan.NewRootStartBytes += MiB

	_, err := ExecuteFreshInstallerResize(
		context.Background(),
		r,
		ExecuteInput{
			Topology:   top,
			Plan:       plan,
			Passphrase: []byte("secret"),
		},
	)
	if err == nil ||
		!strings.Contains(err.Error(), "move root partition start") {
		t.Fatalf("error=%v", err)
	}

	if len(r.runs) != 0 {
		t.Fatalf("mutation commands ran: %v", r.runs)
	}
}

func TestExecutorRejectsIdentityMismatchBeforeMutation(t *testing.T) {
	top, plan, r := executorFixture(t)

	key := commandKey(
		"lsblk",
		"-dnro",
		"PTUUID",
		top.DiskPath,
	)
	r.outputs[key] = []byte("different-guid\n")

	_, err := ExecuteFreshInstallerResize(
		context.Background(),
		r,
		ExecuteInput{
			Topology:   top,
			Plan:       plan,
			Passphrase: []byte("secret"),
		},
	)
	if err == nil ||
		!strings.Contains(err.Error(), "topology changed") {
		t.Fatalf("error=%v", err)
	}

	if len(r.runs) != 0 {
		t.Fatalf("mutation commands ran: %v", r.runs)
	}
}

func TestGPTUsesInclusiveEndSectorWithoutMovingStart(t *testing.T) {
	top, plan, r := executorFixture(t)

	_, err := ExecuteFreshInstallerResize(
		context.Background(),
		r,
		ExecuteInput{
			Topology:   top,
			Plan:       plan,
			Passphrase: []byte("secret"),
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	wantEnd := plan.NewRootEndBytes/
		top.LogicalSectorBytes - 1

	var parted string
	for _, call := range r.inputRuns {
		joinedCall := strings.Join(call, " ")
		if strings.Contains(joinedCall, " parted ") &&
			strings.Contains(joinedCall, " resizepart ") {
			parted = joinedCall
		}
	}

	if parted == "" {
		t.Fatal("parted resizepart command missing")
	}
	if !strings.Contains(
		parted,
		fmt.Sprintf("resizepart 2 %ds", wantEnd),
	) {
		t.Fatalf(
			"wrong resizepart boundary:\n%s",
			parted,
		)
	}
}

func TestExecutorDoesNotUseDestructivePartitionRecreation(t *testing.T) {
	top, plan, r := executorFixture(t)

	_, err := ExecuteFreshInstallerResize(
		context.Background(),
		r,
		ExecuteInput{
			Topology:   top,
			Plan:       plan,
			Passphrase: []byte("secret"),
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	joined := ""
	for _, call := range r.runs {
		joined += strings.Join(call, " ") + "\n"
	}

	for _, forbidden := range []string{
		"sgdisk --delete",
		"sgdisk --new",
		"mkfs",
		"wipefs",
		"luksFormat",
		"resize2fs",
		"cryptsetup resize",
	} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf(
				"forbidden executor operation %q:\n%s",
				forbidden,
				joined,
			)
		}
	}
}
