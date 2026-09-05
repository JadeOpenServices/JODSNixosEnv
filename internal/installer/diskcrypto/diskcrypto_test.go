package diskcrypto

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMappingAndTPMConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hardware.nix")
	initial := "{\n  boot.initrd.luks.devices.\"cryptroot\".device = \"/dev/disk/by-uuid/example\";\n}\n"
	if err := os.WriteFile(path, []byte(initial), 0644); err != nil {
		t.Fatal(err)
	}
	mapping, err := Mapping(path)
	if err != nil || mapping != "cryptroot" {
		t.Fatalf("%q %v", mapping, err)
	}
	details, err := MappingDetails(path)
	if err != nil || details.Device != "/dev/disk/by-uuid/example" {
		t.Fatalf("unexpected mapping details: %#v %v", details, err)
	}
	if err := EnableTPMConfig(path, mapping); err != nil {
		t.Fatal(err)
	}
	if err := EnableTPMConfig(path, mapping); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if strings.Count(string(data), "boot.initrd.systemd.tpm2.enable") != 1 {
		t.Fatalf("not idempotent:\n%s", data)
	}
}

func TestGenerateKey(t *testing.T) {
	a, err := GenerateKey()
	if err != nil || len(a) != 64 {
		t.Fatalf("%q %v", a, err)
	}
	b, _ := GenerateKey()
	if a == b {
		t.Fatal("duplicate random keys")
	}
}

func TestParseLUKSDevices(t *testing.T) {
	devices := parseLUKSDevices("/dev/nvme0n1p2 crypto_LUKS\n/dev/nvme1n1p2 crypto_LUKS\n/dev/nvme0n1p1 vfat\n")
	if len(devices) != 2 || devices[0] != "/dev/nvme0n1p2" || devices[1] != "/dev/nvme1n1p2" {
		t.Fatalf("unexpected devices: %#v", devices)
	}
}

func TestTPMEnrollmentCannotBecomeUnbound(t *testing.T) {
	args := strings.Join(enrollmentArgs("/dev/disk/by-uuid/test", "/run/key"), " ")
	if !strings.Contains(args, "--tpm2-pcrlock=/var/lib/systemd/pcrlock.json") {
		t.Fatalf("TPM enrollment lacks measured-boot policy: %s", args)
	}
	if strings.Contains(args, "--tpm2-pcrs=0") || strings.Contains(args, "--tpm2-pcrs=\"") {
		t.Fatalf("TPM enrollment contains an unbound PCR policy: %s", args)
	}
}
