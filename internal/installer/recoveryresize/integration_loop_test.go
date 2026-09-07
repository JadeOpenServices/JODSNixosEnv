//go:build integration

package recoveryresize

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const (
	integrationImageBytes = uint64(8 * 1024 * 1024 * 1024)
	integrationMiB        = uint64(1024 * 1024)
)

var loopDevicePattern = regexp.MustCompile(`^/dev/loop[0-9]+$`)

type integrationRunner struct{}

func executable(name string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", fmt.Errorf("%s is unavailable: %w", name, err)
	}
	return path, nil
}

func privilegedCommand(
	ctx context.Context,
	name string,
	args ...string,
) (*exec.Cmd, error) {
	// Executor mutation calls intentionally contain "sudo". Avoid nesting
	// sudo and resolve the real executable from the integration environment.
	if name == "sudo" {
		if len(args) == 0 {
			return nil, fmt.Errorf("sudo command is empty")
		}
		name = args[0]
		args = args[1:]
	}

	path, err := executable(name)
	if err != nil {
		return nil, err
	}

	sudoPath, err := executable("sudo")
	if err != nil {
		return nil, err
	}

	allArgs := []string{"-n", "--", path}
	allArgs = append(allArgs, args...)

	return exec.CommandContext(
		ctx,
		sudoPath,
		allArgs...,
	), nil
}

func (integrationRunner) Output(
	ctx context.Context,
	name string,
	args ...string,
) ([]byte, error) {
	cmd, err := privilegedCommand(ctx, name, args...)
	if err != nil {
		return nil, err
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf(
			"%s %v: %w: %s",
			name,
			args,
			err,
			strings.TrimSpace(string(out)),
		)
	}
	return out, nil
}

func (integrationRunner) Run(
	ctx context.Context,
	name string,
	args ...string,
) error {
	cmd, err := privilegedCommand(ctx, name, args...)
	if err != nil {
		return err
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf(
			"%s %v: %w: %s",
			name,
			args,
			err,
			strings.TrimSpace(string(out)),
		)
	}
	return nil
}

func (integrationRunner) RunInput(
	ctx context.Context,
	input []byte,
	name string,
	args ...string,
) error {
	cmd, err := privilegedCommand(ctx, name, args...)
	if err != nil {
		return err
	}

	cmd.Stdin = bytes.NewReader(input)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf(
			"%s %v: %w: %s",
			name,
			args,
			err,
			strings.TrimSpace(string(out)),
		)
	}
	return nil
}

func requireLoopBackedBy(
	ctx context.Context,
	r integrationRunner,
	loopDevice string,
	image string,
) error {
	if !loopDevicePattern.MatchString(loopDevice) {
		return fmt.Errorf(
			"refusing non-loop integration device %q",
			loopDevice,
		)
	}

	kindRaw, err := r.Output(
		ctx,
		"lsblk",
		"-dnro",
		"TYPE",
		loopDevice,
	)
	if err != nil {
		return err
	}
	if strings.TrimSpace(string(kindRaw)) != "loop" {
		return fmt.Errorf(
			"refusing device %q: lsblk type is not loop",
			loopDevice,
		)
	}

	backRaw, err := r.Output(
		ctx,
		"losetup",
		"--list",
		"--noheadings",
		"--raw",
		"--output",
		"BACK-FILE",
		loopDevice,
	)
	if err != nil {
		return err
	}

	want, err := filepath.EvalSymlinks(image)
	if err != nil {
		return fmt.Errorf("resolve image path: %w", err)
	}

	got := strings.TrimSpace(string(backRaw))
	got, err = filepath.EvalSymlinks(got)
	if err != nil {
		return fmt.Errorf(
			"resolve loop backing path %q: %w",
			got,
			err,
		)
	}

	if got != want {
		return fmt.Errorf(
			"refusing loop device with wrong backing file: got=%q want=%q",
			got,
			want,
		)
	}

	return nil
}

func partitionByNumber(
	ctx context.Context,
	r integrationRunner,
	disk string,
	number uint64,
) (string, error) {
	raw, err := r.Output(
		ctx,
		"lsblk",
		"-lnpo",
		"NAME,TYPE,PARTN",
		disk,
	)
	if err != nil {
		return "", err
	}

	want := strconv.FormatUint(number, 10)

	for _, line := range strings.Split(string(raw), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 3 {
			continue
		}
		if fields[1] == "part" && fields[2] == want {
			if !strings.HasPrefix(fields[0], disk) {
				return "", fmt.Errorf(
					"partition %q escaped loop device %q",
					fields[0],
					disk,
				)
			}
			return fields[0], nil
		}
	}

	return "", fmt.Errorf(
		"partition %d not found on %s",
		number,
		disk,
	)
}

func TestLoopBackedEncryptedBtrfsResize(t *testing.T) {
	if os.Getenv("GJALLAR_RUN_LOOP_RESIZE_TEST") != "1" {
		t.Skip(
			"set GJALLAR_RUN_LOOP_RESIZE_TEST=1 for destructive loop-file integration test",
		)
	}

	ctx := context.Background()
	r := integrationRunner{}

	// This test deliberately uses the production executor's required /mnt.
	// Refuse to interfere with anything already mounted there.
	if out, err := exec.Command(
		"findmnt",
		"-n",
		"--mountpoint",
		FreshInstallerTargetMountpoint,
	).CombinedOutput(); err == nil &&
		strings.TrimSpace(string(out)) != "" {
		t.Fatalf(
			"%s is already mounted; refusing integration test",
			FreshInstallerTargetMountpoint,
		)
	}

	imageFile, err := os.CreateTemp(
		"/var/tmp",
		"gjallar-gjal65-loop-*.img",
	)
	if err != nil {
		t.Fatal(err)
	}
	image := imageFile.Name()

	if err := imageFile.Close(); err != nil {
		t.Fatal(err)
	}

	if err := os.Truncate(
		image,
		int64(integrationImageBytes),
	); err != nil {
		_ = os.Remove(image)
		t.Fatal(err)
	}

	// Confirm this is exactly the temporary regular file we created.
	info, err := os.Lstat(image)
	if err != nil {
		_ = os.Remove(image)
		t.Fatal(err)
	}
	if !info.Mode().IsRegular() {
		_ = os.Remove(image)
		t.Fatalf(
			"integration backing path is not a regular file: %s",
			image,
		)
	}

	var loopDevice string
	var partition string

	mapperName := fmt.Sprintf(
		"gjallar-gjal65-test-%d",
		os.Getpid(),
	)
	mapperPath := "/dev/mapper/" + mapperName

	passphrase := []byte(
		"gjallar-gjal65-loop-integration-only",
	)

	cleanup := func() {
		// Cleanup is intentionally constrained to the known test mapper,
		// /mnt, and the verified loop device.
		_ = r.Run(
			context.Background(),
			"umount",
			FreshInstallerTargetMountpoint,
		)

		_ = r.Run(
			context.Background(),
			"cryptsetup",
			"close",
			mapperName,
		)

		if loopDevice != "" &&
			loopDevicePattern.MatchString(loopDevice) {
			if err := requireLoopBackedBy(
				context.Background(),
				r,
				loopDevice,
				image,
			); err == nil {
				_ = r.Run(
					context.Background(),
					"losetup",
					"--detach",
					loopDevice,
				)
				_ = r.Run(
					context.Background(),
					"udevadm",
					"settle",
				)
			}
		}

		_ = os.Remove(image)
	}
	defer cleanup()

	// Allocate a loop device ONLY from our freshly-created regular file.
	rawLoop, err := r.Output(
		ctx,
		"losetup",
		"--find",
		"--show",
		"--partscan",
		"--nooverlap",
		image,
	)
	if err != nil {
		t.Fatal(err)
	}

	loopDevice = strings.TrimSpace(string(rawLoop))

	if err := requireLoopBackedBy(
		ctx,
		r,
		loopDevice,
		image,
	); err != nil {
		t.Fatal(err)
	}

	// 8 GiB fake disk:
	//
	// p1  1 MiB .. 513 MiB    fake ESP placeholder
	// p2 513 MiB .. 7681 MiB  encrypted Btrfs root
	//
	// No physical disk path can enter this block: loopDevice has already
	// passed both TYPE=loop and exact backing-file verification.
	if err := r.Run(
		ctx,
		"parted",
		"--script",
		"--align",
		"optimal",
		loopDevice,
		"mklabel",
		"gpt",
		"mkpart",
		"ESP",
		"fat32",
		"1MiB",
		"513MiB",
		"set",
		"1",
		"esp",
		"on",
		"mkpart",
		"root",
		"513MiB",
		"7681MiB",
	); err != nil {
		t.Fatal(err)
	}

	if err := r.Run(ctx, "partprobe", loopDevice); err != nil {
		t.Fatal(err)
	}
	if err := r.Run(ctx, "udevadm", "settle"); err != nil {
		t.Fatal(err)
	}

	partition, err = partitionByNumber(
		ctx,
		r,
		loopDevice,
		2,
	)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(partition, loopDevice) {
		t.Fatalf(
			"refusing partition outside loop device: %s",
			partition,
		)
	}

	// Re-verify the loop backing immediately before the first destructive
	// operation on its partition.
	if err := requireLoopBackedBy(
		ctx,
		r,
		loopDevice,
		image,
	); err != nil {
		t.Fatal(err)
	}

	if err := r.RunInput(
		ctx,
		passphrase,
		"cryptsetup",
		"luksFormat",
		"--type",
		"luks2",
		"--batch-mode",
		"--key-file",
		"-",
		partition,
	); err != nil {
		t.Fatal(err)
	}

	if err := r.RunInput(
		ctx,
		passphrase,
		"cryptsetup",
		"open",
		"--type",
		"luks2",
		"--key-file",
		"-",
		partition,
		mapperName,
	); err != nil {
		t.Fatal(err)
	}

	if err := r.Run(
		ctx,
		"mkfs.btrfs",
		"-f",
		"-L",
		"GJALLAR-LOOP-TEST",
		mapperPath,
	); err != nil {
		t.Fatal(err)
	}

	if err := r.Run(
		ctx,
		"mkdir",
		"-p",
		FreshInstallerTargetMountpoint,
	); err != nil {
		t.Fatal(err)
	}

	if err := r.Run(
		ctx,
		"mount",
		mapperPath,
		FreshInstallerTargetMountpoint,
	); err != nil {
		t.Fatal(err)
	}

	sentinelPath := filepath.Join(
		FreshInstallerTargetMountpoint,
		"gjallar-gjal65-sentinel",
	)
	sentinel := []byte(
		"GJAL-65 encrypted Btrfs resize survived\n",
	)

	// /mnt is a freshly-created root filesystem and is therefore owned by
	// root. Write the sentinel through the already constrained privileged
	// integration runner rather than assuming the invoking user owns /mnt.
	if err := r.RunInput(
		ctx,
		sentinel,
		"tee",
		sentinelPath,
	); err != nil {
		t.Fatalf(
			"write sentinel through privileged test runner: %v",
			err,
		)
	}

	if err := r.Run(
		ctx,
		"chmod",
		"0600",
		sentinelPath,
	); err != nil {
		t.Fatalf(
			"set sentinel permissions: %v",
			err,
		)
	}

	if err := r.Run(ctx, "sync"); err != nil {
		t.Fatal(err)
	}

	before, err := DiscoverTopology(
		ctx,
		r,
		DiscoveryInput{
			RootMountpoint:            FreshInstallerTargetMountpoint,
			ExpectedDiskPath:          loopDevice,
			ExpectedRootPartitionPath: partition,
			ExpectedMappingPath:       mapperPath,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if before.RootPartitionNumber != 2 {
		t.Fatalf(
			"unexpected root partition number %d",
			before.RootPartitionNumber,
		)
	}

	req := Requirements{
		RecoveryBytes:           512 * integrationMiB,
		SafetyMarginBytes:       256 * integrationMiB,
		FilesystemHeadroomBytes: 512 * integrationMiB,
		AlignmentBytes:          integrationMiB,
	}

	plan, err := BuildPlan(before, req)
	if err != nil {
		t.Fatal(err)
	}

	if plan.NewRootStartBytes != before.RootStartBytes {
		t.Fatal("planner attempted to move root start")
	}

	// FINAL guard before calling the ACTUAL production executor.
	if err := requireLoopBackedBy(
		ctx,
		r,
		loopDevice,
		image,
	); err != nil {
		t.Fatal(err)
	}

	result, err := ExecuteFreshInstallerResize(
		ctx,
		r,
		ExecuteInput{
			Topology:   before,
			Plan:       plan,
			Passphrase: passphrase,
			Out:        os.Stdout,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if result.RootPartitionSizeBytes !=
		plan.NewRootSizeBytes {
		t.Fatalf(
			"executor result root size=%d expected=%d",
			result.RootPartitionSizeBytes,
			plan.NewRootSizeBytes,
		)
	}

	after, err := DiscoverTopology(
		ctx,
		r,
		DiscoveryInput{
			RootMountpoint:            FreshInstallerTargetMountpoint,
			ExpectedDiskPath:          loopDevice,
			ExpectedRootPartitionPath: partition,
			ExpectedMappingPath:       mapperPath,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if after.DiskGUID != before.DiskGUID {
		t.Fatalf(
			"disk GUID changed: %s -> %s",
			before.DiskGUID,
			after.DiskGUID,
		)
	}
	if after.RootPARTUUID != before.RootPARTUUID {
		t.Fatalf(
			"PARTUUID changed: %s -> %s",
			before.RootPARTUUID,
			after.RootPARTUUID,
		)
	}
	if after.RootPartitionTypeGUID !=
		before.RootPartitionTypeGUID {
		t.Fatalf(
			"partition type changed: %s -> %s",
			before.RootPartitionTypeGUID,
			after.RootPartitionTypeGUID,
		)
	}
	if after.RootStartBytes != before.RootStartBytes {
		t.Fatalf(
			"root start moved: %d -> %d",
			before.RootStartBytes,
			after.RootStartBytes,
		)
	}
	if after.RootSizeBytes != plan.NewRootSizeBytes {
		t.Fatalf(
			"root size=%d expected=%d",
			after.RootSizeBytes,
			plan.NewRootSizeBytes,
		)
	}
	if after.LUKSUUID != before.LUKSUUID {
		t.Fatalf(
			"LUKS UUID changed: %s -> %s",
			before.LUKSUUID,
			after.LUKSUUID,
		)
	}
	if after.BtrfsUUID != before.BtrfsUUID {
		t.Fatalf(
			"Btrfs UUID changed: %s -> %s",
			before.BtrfsUUID,
			after.BtrfsUUID,
		)
	}

	gotSentinel, err := r.Output(
		ctx,
		"cat",
		sentinelPath,
	)
	if err != nil {
		t.Fatalf(
			"sentinel file did not survive resize: %v",
			err,
		)
	}
	if !bytes.Equal(gotSentinel, sentinel) {
		t.Fatalf(
			"sentinel changed: got=%q expected=%q",
			gotSentinel,
			sentinel,
		)
	}

	t.Logf("loop device: %s", loopDevice)
	t.Logf(
		"root bytes: %d -> %d",
		before.RootSizeBytes,
		after.RootSizeBytes,
	)
	t.Logf("disk GUID preserved: %s", after.DiskGUID)
	t.Logf("PARTUUID preserved: %s", after.RootPARTUUID)
	t.Logf("LUKS UUID preserved: %s", after.LUKSUUID)
	t.Logf("Btrfs UUID preserved: %s", after.BtrfsUUID)
	t.Log("sentinel data survived")
}
