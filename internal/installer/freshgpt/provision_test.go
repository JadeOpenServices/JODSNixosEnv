package freshgpt

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/bakanura/gjallarOS/internal/installer/diskplan"
)

type fakeRunner struct {
	calls   [][]string
	outputs map[string][]byte
}

func key(name string, args ...string) string {
	return strings.Join(append([]string{name}, args...), "\x00")
}

func (r *fakeRunner) Run(
	_ context.Context,
	name string,
	args ...string,
) error {
	r.calls = append(
		r.calls,
		append([]string{name}, args...),
	)
	return nil
}

func (r *fakeRunner) Output(
	_ context.Context,
	name string,
	args ...string,
) ([]byte, error) {
	call := append([]string{name}, args...)
	r.calls = append(r.calls, call)

	out, ok := r.outputs[key(name, args...)]
	if !ok {
		return nil, fmt.Errorf("unexpected command: %v", call)
	}
	return out, nil
}

func plan() diskplan.Plan {
	recovery := &diskplan.Partition{
		Role:      diskplan.RoleRecovery,
		Number:    3,
		Label:     "JODS-RECOVERY",
		TypeGUID:  "bc13c2ff-59e6-4262-a352-b275fd6f7172",
		PARTUUID:  "99999999-8888-7777-6666-555555555555",
		SizeBytes: 12 * 1024 * 1024 * 1024,
		Filesystem: diskplan.Filesystem{
			Type:  "vfat",
			Label: "JODS-RECOVERY",
		},
		MountPoint: "/recovery",
	}

	return diskplan.Plan{
		SchemaVersion: diskplan.SchemaVersion,
		TargetDisk: diskplan.Disk{
			Path:        "/dev/nvme1n1",
			Model:       "Test NVMe",
			Serial:      "SERIAL",
			WWN:         "eui.0011223344556677",
			SizeBytes:   10485760000,
			GPTDiskGUID: "11111111-2222-4333-8444-555555555555",
		},
		ESP: diskplan.Partition{
			Role:      diskplan.RoleESP,
			Number:    1,
			Label:     "EFI",
			TypeGUID:  "c12a7328-f81f-11d2-ba4b-00a0c93ec93b",
			PARTUUID:  "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee",
			SizeBytes: 1024 * 1024 * 1024,
			Filesystem: diskplan.Filesystem{
				Type:  "vfat",
				Label: "EFI",
			},
			MountPoint: "/boot",
		},
		Root: diskplan.Root{
			Partition: diskplan.Partition{
				Role:      diskplan.RoleRoot,
				Number:    2,
				Label:     "GJALLAROS",
				TypeGUID:  "ca7d7ccb-63ed-4c53-861c-1742536059cc",
				PARTUUID:  "11111111-aaaa-4bbb-8ccc-222222222222",
				SizeBytes: 6 * 1024 * 1024 * 1024,
				Filesystem: diskplan.Filesystem{
					Type:  "btrfs",
					Label: "GJALLAROS",
				},
				MountPoint: "/",
			},
			Encryption: diskplan.Encryption{
				Type:        diskplan.EncryptionLUKS2,
				MappingName: "cryptroot",
			},
		},
		Recovery: recovery,
	}
}

func TestProvisionCreatesOnlyCanonicalPartitions(t *testing.T) {
	p := plan()

	lsblk := fmt.Sprintf(`{
	  "blockdevices": [{
	    "path": %q,
	    "type": "disk",
	    "size": %d,
	    "fstype": null,
	    "pttype": "gpt",
	    "ptuuid": %q,
	    "children": [
	      {
	        "path": "/dev/nvme1n1p1",
	        "type": "part",
	        "size": %d,
	        "fstype": null,
	        "partn": 1,
	        "partlabel": "EFI",
	        "parttype": %q,
	        "partuuid": %q
	      },
	      {
	        "path": "/dev/nvme1n1p2",
	        "type": "part",
	        "size": %d,
	        "fstype": null,
	        "partn": 2,
	        "partlabel": "GJALLAROS",
	        "parttype": %q,
	        "partuuid": %q
	      },
	      {
	        "path": "/dev/nvme1n1p3",
	        "type": "part",
	        "size": %d,
	        "fstype": null,
	        "partn": 3,
	        "partlabel": "JODS-RECOVERY",
	        "parttype": %q,
	        "partuuid": %q
	      }
	    ]
	  }]
	}`,
		p.TargetDisk.Path,
		p.TargetDisk.SizeBytes,
		p.TargetDisk.GPTDiskGUID,
		p.ESP.SizeBytes,
		p.ESP.TypeGUID,
		p.ESP.PARTUUID,
		p.Root.Partition.SizeBytes,
		p.Root.Partition.TypeGUID,
		p.Root.Partition.PARTUUID,
		p.Recovery.SizeBytes,
		p.Recovery.TypeGUID,
		p.Recovery.PARTUUID,
	)

	r := &fakeRunner{
		outputs: map[string][]byte{
			key(
				"lsblk",
				"-J",
				"-b",
				"-p",
				"-o",
				"PATH,TYPE,SIZE,FSTYPE,PTTYPE,PTUUID,PARTN,PARTLABEL,PARTTYPE,PARTUUID",
				"--",
				p.TargetDisk.Path,
			): []byte(lsblk),
		},
	}

	var out strings.Builder
	result, err := provision(
		context.Background(),
		Input{Plan: p, Out: &out},
		r,
	)
	if err != nil {
		t.Fatal(err)
	}

	if result.RootPARTUUID != p.Root.Partition.PARTUUID {
		t.Fatalf("unexpected result: %#v", result)
	}

	all := make([]string, 0, len(r.calls))
	for _, call := range r.calls {
		all = append(all, strings.Join(call, " "))
	}
	joined := strings.Join(all, "\n")

	for _, required := range []string{
		"sudo sgdisk --zap-all /dev/nvme1n1",
		"sudo sgdisk --disk-guid=" + p.TargetDisk.GPTDiskGUID,
		"--new=1:0:+1024M",
		"--new=2:0:+6144M",
		"--new=3:0:+12288M",
		"--partition-guid=1:" + strings.ToLower(p.ESP.PARTUUID),
		"--partition-guid=2:" + strings.ToLower(p.Root.Partition.PARTUUID),
		"--partition-guid=3:" + strings.ToLower(p.Recovery.PARTUUID),
		"sudo partprobe /dev/nvme1n1",
		"sudo udevadm settle",
	} {
		if !strings.Contains(joined, required) {
			t.Fatalf(
				"missing command fragment %q\ncalls:\n%s",
				required,
				joined,
			)
		}
	}

	for _, forbidden := range []string{
		"mkfs",
		"cryptsetup",
		"mount ",
		"wipefs",
	} {
		if strings.Contains(joined, forbidden) {
			t.Fatalf(
				"fresh GPT stage performed forbidden operation %q:\n%s",
				forbidden,
				joined,
			)
		}
	}
}

func TestProvisionRejectsFilesystemAlreadyPresent(t *testing.T) {
	p := plan()

	lsblk := fmt.Sprintf(`{
	  "blockdevices": [{
	    "path": %q,
	    "type": "disk",
	    "size": %d,
	    "pttype": "gpt",
	    "ptuuid": %q,
	    "children": [
	      {
	        "type":"part",
	        "size":%d,
	        "fstype":"vfat",
	        "partn":1,
	        "partlabel":"EFI",
	        "parttype":%q,
	        "partuuid":%q
	      },
	      {
	        "type":"part",
	        "size":%d,
	        "fstype":null,
	        "partn":2,
	        "partlabel":"GJALLAROS",
	        "parttype":%q,
	        "partuuid":%q
	      },
	      {
	        "type":"part",
	        "size":%d,
	        "fstype":null,
	        "partn":3,
	        "partlabel":"JODS-RECOVERY",
	        "parttype":%q,
	        "partuuid":%q
	      }
	    ]
	  }]
	}`,
		p.TargetDisk.Path,
		p.TargetDisk.SizeBytes,
		p.TargetDisk.GPTDiskGUID,
		p.ESP.SizeBytes,
		p.ESP.TypeGUID,
		p.ESP.PARTUUID,
		p.Root.Partition.SizeBytes,
		p.Root.Partition.TypeGUID,
		p.Root.Partition.PARTUUID,
		p.Recovery.SizeBytes,
		p.Recovery.TypeGUID,
		p.Recovery.PARTUUID,
	)

	r := &fakeRunner{
		outputs: map[string][]byte{
			key(
				"lsblk",
				"-J",
				"-b",
				"-p",
				"-o",
				"PATH,TYPE,SIZE,FSTYPE,PTTYPE,PTUUID,PARTN,PARTLABEL,PARTTYPE,PARTUUID",
				"--",
				p.TargetDisk.Path,
			): []byte(lsblk),
		},
	}

	var out strings.Builder
	_, err := provision(
		context.Background(),
		Input{Plan: p, Out: &out},
		r,
	)
	if err == nil {
		t.Fatal("verification accepted unexpected filesystem")
	}
}
