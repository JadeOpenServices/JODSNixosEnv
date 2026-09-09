package diskcrypto

import (
	"encoding/json"
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

func TestReadTPM2TokenID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metadata.json")
	doc := map[string]any{
		"tokens": map[string]any{
			"7": map[string]any{
				"type":     "systemd-tpm2",
				"keyslots": []string{"1"},
			},
		},
	}

	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}

	id, err := ReadTPM2TokenID(path)
	if err != nil {
		t.Fatal(err)
	}
	if id != "7" {
		t.Fatalf("token id = %q, want 7", id)
	}
}

func TestReadTPM2TokenIDRejectsMultipleTokens(t *testing.T) {
	path := filepath.Join(t.TempDir(), "metadata.json")
	raw := []byte(`{"tokens":{"1":{"type":"systemd-tpm2"},"2":{"type":"systemd-tpm2"}}}`)

	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := ReadTPM2TokenID(path); err == nil {
		t.Fatal("accepted multiple TPM2 tokens")
	}
}

func TestWriteTPM2KeyslotRecord(t *testing.T) {
	dir := t.TempDir()
	metadataPath := filepath.Join(dir, "metadata.json")
	targetPath := filepath.Join(dir, "keyslots.json")

	raw := []byte(`{"tokens":{"9":{"type":"systemd-tpm2","keyslots":["3"]}}}`)
	if err := os.WriteFile(metadataPath, raw, 0600); err != nil {
		t.Fatal(err)
	}

	if err := WriteTPM2KeyslotRecord(
		metadataPath,
		targetPath,
		"/dev/disk/by-uuid/test",
		"/var/lib/systemd/pcrlock.json",
	); err != nil {
		t.Fatal(err)
	}

	out, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}

	var record TPM2KeyslotRecord
	if err := json.Unmarshal(out, &record); err != nil {
		t.Fatal(err)
	}

	if record.Schema != 1 {
		t.Fatalf("schema = %d, want 1", record.Schema)
	}
	if record.Device != "/dev/disk/by-uuid/test" {
		t.Fatalf("device = %q", record.Device)
	}
	if record.Policy != "/var/lib/systemd/pcrlock.json" {
		t.Fatalf("policy = %q", record.Policy)
	}
	if !record.HumanRecoveryVerified {
		t.Fatal("humanRecoveryVerified = false")
	}
	if got := record.TPM2Tokens["9"]; len(got) != 1 || got[0] != "3" {
		t.Fatalf("TPM2 token keyslots = %#v", got)
	}
}
