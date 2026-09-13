package recoveryresize

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

type discoveryRunner struct {
	outputs map[string][]byte
	calls   [][]string
}

func commandKey(name string, args ...string) string {
	return strings.Join(
		append([]string{name}, args...),
		"\x00",
	)
}

func (r *discoveryRunner) Output(
	_ context.Context,
	name string,
	args ...string,
) ([]byte, error) {
	call := append([]string{name}, args...)
	r.calls = append(r.calls, call)

	key := commandKey(name, args...)
	out, ok := r.outputs[key]
	if !ok {
		return nil, fmt.Errorf(
			"unexpected discovery command: %v",
			call,
		)
	}

	return out, nil
}

func validDiscoveryRunner() *discoveryRunner {
	disk := "/dev/nvme0n1"
	part := "/dev/nvme0n1p2"
	mapping := "/dev/mapper/cryptroot"
	mountpoint := "/mnt"

	startSectors := uint64(2097152)
	sectorSize := uint64(512)
	rootSize := 900 * GiB
	payloadOffset := 16 * MiB

	return &discoveryRunner{
		outputs: map[string][]byte{
			commandKey(
				"findmnt",
				"-nro",
				"SOURCE,FSTYPE,OPTIONS",
				"--target",
				mountpoint,
			): []byte(mapping + " btrfs rw,relatime,ssd\n"),

			commandKey(
				"cryptsetup",
				"status",
				"cryptroot",
			): []byte(`
/dev/mapper/cryptroot is active and is in use.
  type:    LUKS2
  cipher:  aes-xts-plain64
  device:  /dev/nvme0n1p2
  sector size:  512
  offset:  32768 sectors
  size:    1887408128 sectors
`),

			commandKey(
				"lsblk",
				"-dnro",
				"TYPE",
				part,
			): []byte("part\n"),

			commandKey(
				"lsblk",
				"-dnro",
				"TYPE",
				disk,
			): []byte("disk\n"),

			commandKey(
				"lsblk",
				"-dnro",
				"PKNAME",
				part,
			): []byte("nvme0n1\n"),

			commandKey(
				"blockdev",
				"--getss",
				disk,
			): []byte(fmt.Sprintf("%d\n", sectorSize)),

			commandKey(
				"blockdev",
				"--getsize64",
				disk,
			): []byte(fmt.Sprintf("%d\n", 1000*GiB)),

			commandKey(
				"blockdev",
				"--getsize64",
				part,
			): []byte(fmt.Sprintf("%d\n", rootSize)),

			commandKey(
				"lsblk",
				"-dnro",
				"START",
				part,
			): []byte(fmt.Sprintf("%d\n", startSectors)),

			commandKey(
				"lsblk",
				"-dnro",
				"PTUUID",
				disk,
			): []byte(
				"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee\n",
			),

			commandKey(
				"lsblk",
				"-dnro",
				"PARTN",
				part,
			): []byte("2\n"),

			commandKey(
				"lsblk",
				"-dnro",
				"PARTTYPE",
				part,
			): []byte(
				"ca7d7ccb-63ed-4c53-861c-1742536059cc\n",
			),

			commandKey(
				"lsblk",
				"-dnro",
				"PARTUUID",
				part,
			): []byte(
				"11111111-2222-3333-4444-555555555555\n",
			),

			commandKey(
				"cryptsetup",
				"luksUUID",
				part,
			): []byte("22222222-3333-4444-5555-666666666666\n"),

			commandKey(
				"cryptsetup",
				"luksDump",
				"--dump-json-metadata",
				part,
			): []byte(fmt.Sprintf(`{
  "segments": {
    "0": {
      "type": "crypt",
      "offset": "%d",
      "size": "dynamic"
    }
  }
}`, payloadOffset)),

			commandKey(
				"btrfs",
				"filesystem",
				"show",
				"--raw",
				mountpoint,
			): []byte(fmt.Sprintf(
				`Label: 'GJALLAROS'  uuid: 33333333-4444-5555-6666-777777777777
	Total devices 1 FS bytes used 429496729600
	devid    1 size %d used %d path %s
`,
				rootSize-payloadOffset,
				400*GiB,
				mapping,
			)),

			commandKey(
				"cat",
				"/sys/fs/btrfs/33333333-4444-5555-6666-777777777777/exclusive_operation",
			): []byte("none\n"),

			commandKey(
				"swapon",
				"--show=NAME",
				"--noheadings",
			): []byte(""),
		},
	}
}

func discoveryInput() DiscoveryInput {
	return DiscoveryInput{
		RootMountpoint:            "/mnt",
		ExpectedDiskPath:          "/dev/nvme0n1",
		ExpectedRootPartitionPath: "/dev/nvme0n1p2",
		ExpectedMappingPath:       "/dev/mapper/cryptroot",
	}
}

func TestDiscoverTopology(t *testing.T) {
	r := validDiscoveryRunner()

	top, err := DiscoverTopology(
		context.Background(),
		r,
		discoveryInput(),
	)
	if err != nil {
		t.Fatal(err)
	}

	if top.RootFilesystem != "btrfs" {
		t.Fatalf("filesystem=%q", top.RootFilesystem)
	}
	if top.DiskGUID !=
		"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee" {
		t.Fatalf("disk GUID=%q", top.DiskGUID)
	}
	if top.RootPARTUUID !=
		"11111111-2222-3333-4444-555555555555" {
		t.Fatalf("PARTUUID=%q", top.RootPARTUUID)
	}
	if top.LUKSUUID !=
		"22222222-3333-4444-5555-666666666666" {
		t.Fatalf("LUKS UUID=%q", top.LUKSUUID)
	}
	if top.BtrfsUUID !=
		"33333333-4444-5555-6666-777777777777" {
		t.Fatalf("Btrfs UUID=%q", top.BtrfsUUID)
	}
	if top.LUKSPayloadOffsetBytes != 16*MiB {
		t.Fatalf(
			"payload offset=%d",
			top.LUKSPayloadOffsetBytes,
		)
	}

	wantStart := uint64(2097152) * 512
	if top.RootStartBytes != wantStart {
		t.Fatalf(
			"root start=%d want=%d",
			top.RootStartBytes,
			wantStart,
		)
	}

	if !top.RootMountedRW {
		t.Fatal("root was not marked read-write")
	}
	if top.SwapActive {
		t.Fatal("empty swapon output marked active")
	}
}

func TestDiscoverRejectsActualExt4BeforeCryptoInspection(t *testing.T) {
	r := validDiscoveryRunner()

	key := commandKey(
		"findmnt",
		"-nro",
		"SOURCE,FSTYPE,OPTIONS",
		"--target",
		"/mnt",
	)
	r.outputs[key] = []byte(
		"/dev/mapper/cryptroot ext4 rw,relatime\n",
	)

	_, err := DiscoverTopology(
		context.Background(),
		r,
		discoveryInput(),
	)
	if err == nil {
		t.Fatal("ext4 topology was accepted")
	}
	if !strings.Contains(
		err.Error(),
		"Current filesystem: ext4. No disk changes were made.",
	) {
		t.Fatalf("unexpected error: %v", err)
	}

	for _, call := range r.calls {
		if len(call) > 0 && call[0] == "cryptsetup" {
			t.Fatalf(
				"crypto inspected after ext4 capability failure: %v",
				call,
			)
		}
	}
}

func TestDiscoverRejectsWrongMountedMapping(t *testing.T) {
	r := validDiscoveryRunner()

	key := commandKey(
		"findmnt",
		"-nro",
		"SOURCE,FSTYPE,OPTIONS",
		"--target",
		"/mnt",
	)
	r.outputs[key] = []byte(
		"/dev/mapper/other btrfs rw,relatime\n",
	)

	_, err := DiscoverTopology(
		context.Background(),
		r,
		discoveryInput(),
	)
	if err == nil ||
		!strings.Contains(err.Error(), "root mapping mismatch") {
		t.Fatalf("error=%v", err)
	}
}

func TestDiscoverRejectsWrongLUKSBackingPartition(t *testing.T) {
	r := validDiscoveryRunner()

	key := commandKey(
		"cryptsetup",
		"status",
		"cryptroot",
	)
	r.outputs[key] = []byte(`
/dev/mapper/cryptroot is active.
  type:    LUKS2
  device:  /dev/nvme9n1p9
`)

	_, err := DiscoverTopology(
		context.Background(),
		r,
		discoveryInput(),
	)
	if err == nil ||
		!strings.Contains(err.Error(), "backing-device mismatch") {
		t.Fatalf("error=%v", err)
	}
}

func TestDiscoverAcceptsLoopStorageContainer(t *testing.T) {
	r := validDiscoveryRunner()

	key := commandKey(
		"lsblk",
		"-dnro",
		"TYPE",
		"/dev/nvme0n1",
	)
	r.outputs[key] = []byte("loop\n")

	top, err := DiscoverTopology(
		context.Background(),
		r,
		discoveryInput(),
	)
	if err != nil {
		t.Fatal(err)
	}

	if top.DiskPath != "/dev/nvme0n1" {
		t.Fatalf("disk=%q", top.DiskPath)
	}
}

func TestDiscoverRejectsUnsupportedContainerType(t *testing.T) {
	for _, deviceType := range []string{
		"rom",
		"dm",
		"crypt",
		"lvm",
	} {
		t.Run(deviceType, func(t *testing.T) {
			r := validDiscoveryRunner()

			key := commandKey(
				"lsblk",
				"-dnro",
				"TYPE",
				"/dev/nvme0n1",
			)
			r.outputs[key] = []byte(deviceType + "\n")

			_, err := DiscoverTopology(
				context.Background(),
				r,
				discoveryInput(),
			)
			if err == nil ||
				!strings.Contains(
					err.Error(),
					"unsupported block type",
				) {
				t.Fatalf("error=%v", err)
			}
		})
	}
}

func TestDiscoverRejectsWrongPhysicalParent(t *testing.T) {
	r := validDiscoveryRunner()

	key := commandKey(
		"lsblk",
		"-dnro",
		"PKNAME",
		"/dev/nvme0n1p2",
	)
	r.outputs[key] = []byte("nvme9n1\n")

	_, err := DiscoverTopology(
		context.Background(),
		r,
		discoveryInput(),
	)
	if err == nil ||
		!strings.Contains(err.Error(), "parent mismatch") {
		t.Fatalf("error=%v", err)
	}
}

func TestDiscoverRejectsReadOnlyBtrfs(t *testing.T) {
	r := validDiscoveryRunner()

	key := commandKey(
		"findmnt",
		"-nro",
		"SOURCE,FSTYPE,OPTIONS",
		"--target",
		"/mnt",
	)
	r.outputs[key] = []byte(
		"/dev/mapper/cryptroot btrfs ro,relatime\n",
	)

	_, err := DiscoverTopology(
		context.Background(),
		r,
		discoveryInput(),
	)
	if err == nil ||
		!strings.Contains(err.Error(), "not mounted read-write") {
		t.Fatalf("error=%v", err)
	}
}

func TestDiscoverRejectsMultipleBtrfsDevices(t *testing.T) {
	raw := `Label: 'GJALLAROS'  uuid: 33333333-4444-5555-6666-777777777777
	Total devices 2 FS bytes used 10
	devid    1 size 1000 used 500 path /dev/mapper/cryptroot
	devid    2 size 1000 used 500 path /dev/mapper/other
`

	_, _, _, _, _, err := parseBtrfsFilesystemShow(raw)
	if err == nil ||
		!strings.Contains(err.Error(), "exactly one Btrfs device") {
		t.Fatalf("error=%v", err)
	}
}

func TestDiscoverActiveSwapFailsTopologyValidation(t *testing.T) {
	r := validDiscoveryRunner()

	key := commandKey(
		"swapon",
		"--show=NAME",
		"--noheadings",
	)
	r.outputs[key] = []byte("/swap/swapfile\n")

	_, err := DiscoverTopology(
		context.Background(),
		r,
		discoveryInput(),
	)
	if err == nil ||
		!strings.Contains(err.Error(), "active swap") {
		t.Fatalf("error=%v", err)
	}
}

func TestDiscoverExclusiveOperationFailsTopologyValidation(t *testing.T) {
	r := validDiscoveryRunner()

	key := commandKey(
		"cat",
		"/sys/fs/btrfs/33333333-4444-5555-6666-777777777777/exclusive_operation",
	)
	r.outputs[key] = []byte("balance\n")

	_, err := DiscoverTopology(
		context.Background(),
		r,
		discoveryInput(),
	)
	if err == nil ||
		!strings.Contains(err.Error(), "exclusive operation") {
		t.Fatalf("error=%v", err)
	}
}

func TestParseLUKSRejectsAmbiguousSegments(t *testing.T) {
	raw := []byte(`{
  "segments": {
    "0": {"type":"crypt","offset":"16777216","size":"dynamic"},
    "1": {"type":"crypt","offset":"33554432","size":"dynamic"}
  }
}`)

	_, err := parseLUKSPayloadOffset(raw)
	if err == nil ||
		!strings.Contains(err.Error(), "exactly one") {
		t.Fatalf("error=%v", err)
	}
}

func TestDiscoveryCommandsAreReadOnly(t *testing.T) {
	r := validDiscoveryRunner()

	_, err := DiscoverTopology(
		context.Background(),
		r,
		discoveryInput(),
	)
	if err != nil {
		t.Fatal(err)
	}

	joined := ""
	for _, call := range r.calls {
		joined += strings.Join(call, " ") + "\n"
	}

	for _, forbidden := range []string{
		" resize",
		" mkfs",
		" --new=",
		" --delete=",
		"wipefs",
		"parted",
		"mount ",
		"umount",
		"luksFormat",
		"cryptsetup open",
		"cryptsetup close",
	} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf(
				"runtime discovery used mutating command %q:\n%s",
				forbidden,
				joined,
			)
		}
	}
}

func TestDiscoverAllowsActiveSiblingSwap(t *testing.T) {
	r := validDiscoveryRunner()

	r.outputs[commandKey(
		"swapon",
		"--show=NAME",
		"--noheadings",
	)] = []byte("/dev/dm-1\n")

	r.outputs[commandKey(
		"lsblk",
		"-s",
		"-nro",
		"PATH",
		"/dev/dm-1",
	)] = []byte(
		"/dev/mapper/cryptswap\n" +
			"/dev/nvme0n1p3\n" +
			"/dev/nvme0n1\n",
	)

	top, err := DiscoverTopology(
		context.Background(),
		r,
		discoveryInput(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if top.SwapActive {
		t.Fatal("unrelated sibling swap marked as root-backed")
	}
}

func TestDiscoverRejectsRootBackedSwap(t *testing.T) {
	r := validDiscoveryRunner()

	r.outputs[commandKey(
		"swapon",
		"--show=NAME",
		"--noheadings",
	)] = []byte("/dev/dm-0\n")

	r.outputs[commandKey(
		"lsblk",
		"-s",
		"-nro",
		"PATH",
		"/dev/dm-0",
	)] = []byte(
		"/dev/mapper/cryptroot\n" +
			"/dev/nvme0n1p2\n" +
			"/dev/nvme0n1\n",
	)

	_, err := DiscoverTopology(
		context.Background(),
		r,
		discoveryInput(),
	)
	if err == nil || !strings.Contains(err.Error(), "active swap") {
		t.Fatalf("error=%v", err)
	}
}

func TestDiscoverRejectsRootSwapfile(t *testing.T) {
	r := validDiscoveryRunner()

	r.outputs[commandKey(
		"swapon",
		"--show=NAME",
		"--noheadings",
	)] = []byte("/mnt/swapfile\n")

	r.outputs[commandKey(
		"findmnt",
		"-nro",
		"SOURCE",
		"--target",
		"/mnt/swapfile",
	)] = []byte("/dev/mapper/cryptroot\n")

	_, err := DiscoverTopology(
		context.Background(),
		r,
		discoveryInput(),
	)
	if err == nil || !strings.Contains(err.Error(), "active swap") {
		t.Fatalf("error=%v", err)
	}
}
