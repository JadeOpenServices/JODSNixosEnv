package diskcrypto

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMappingAndTPMConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hardware.nix")
	initial := "{\n  boot.initrd.luks.devices.\"cryptroot\".device = \"/dev/disk/x\";\n}\n"
	if err := os.WriteFile(path, []byte(initial), 0644); err != nil {
		t.Fatal(err)
	}
	mapping, err := Mapping(path)
	if err != nil || mapping != "cryptroot" {
		t.Fatalf("%q %v", mapping, err)
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
