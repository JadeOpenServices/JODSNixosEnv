package secureboot

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/JadeOpenServices/gjallarOS/internal/installer/oddc"
)

func frameworkEffectivePolicy() oddc.EffectiveSecureBootFirmwarePolicy {
	return oddc.EffectiveSecureBootFirmwarePolicy{
		SourceEntity: "vendor/framework",
		Policy: oddc.SecureBootFirmwarePolicy{
			Supported:               true,
			FirmwareName:            "Framework UEFI",
			SetupModeStrategy:       "clear-platform-key",
			EnrollmentBackend:       "sbctl",
			RequiredPresent:         []string{"KEK", "db", "dbx"},
			RequiredAbsent:          []string{"PK"},
			PreserveFirmwareBuiltin: []string{"KEK", "db"},
			Untouched:               []string{"dbx"},
			FactoryOwnershipProof:   "pk-equals-pkdefault",
			Instructions: []string{
				"Delete only PK.",
				"Keep KEK, db, and dbx intact.",
			},
		},
	}
}

func TestSnapshotFirmwarePolicyCapturesDetectedPolicy(t *testing.T) {
	snapshot, err := SnapshotFirmwarePolicy(
		"model/framework/laptop-13-amd-ryzen-7040",
		frameworkEffectivePolicy(),
	)
	if err != nil {
		t.Fatal(err)
	}

	if snapshot.ModelID != "model/framework/laptop-13-amd-ryzen-7040" {
		t.Fatalf("ModelID=%q", snapshot.ModelID)
	}

	if snapshot.SourceEntity != "vendor/framework" {
		t.Fatalf("SourceEntity=%q", snapshot.SourceEntity)
	}

	if snapshot.SetupModeStrategy != "clear-platform-key" {
		t.Fatalf("SetupModeStrategy=%q", snapshot.SetupModeStrategy)
	}

	if len(snapshot.PreserveFirmwareBuiltin) != 2 ||
		snapshot.PreserveFirmwareBuiltin[0] != "KEK" ||
		snapshot.PreserveFirmwareBuiltin[1] != "db" {
		t.Fatalf(
			"PreserveFirmwareBuiltin=%v",
			snapshot.PreserveFirmwareBuiltin,
		)
	}
}

func TestSnapshotFirmwarePolicyRejectsUnsupportedPolicy(t *testing.T) {
	_, err := SnapshotFirmwarePolicy(
		"model/hp/zbook-x2-g4",
		oddc.EffectiveSecureBootFirmwarePolicy{
			SourceEntity: "vendor/hp",
			Policy: oddc.SecureBootFirmwarePolicy{
				Supported:         false,
				SetupModeStrategy: "unsupported",
				UnsupportedReason: "Not validated.",
			},
		},
	)

	if err == nil {
		t.Fatal("unsupported policy was snapshotted")
	}
}

func TestFirmwarePolicySnapshotRoundTrip(t *testing.T) {
	snapshot, err := SnapshotFirmwarePolicy(
		"model/framework/laptop-13-amd-ryzen-7040",
		frameworkEffectivePolicy(),
	)
	if err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "firmware-policy.json")

	if err := WriteFirmwarePolicySnapshot(path, snapshot); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}

	if info.Mode().Perm() != 0600 {
		t.Fatalf("mode=%#o", info.Mode().Perm())
	}

	loaded, err := LoadFirmwarePolicySnapshot(path)
	if err != nil {
		t.Fatal(err)
	}

	if loaded.ModelID != snapshot.ModelID ||
		loaded.SourceEntity != snapshot.SourceEntity ||
		loaded.FirmwareName != snapshot.FirmwareName {
		t.Fatalf("loaded snapshot differs: %+v", loaded)
	}
}

func TestFirmwarePolicySnapshotRejectsUnknownField(t *testing.T) {
	path := filepath.Join(t.TempDir(), "firmware-policy.json")

	data := `{
	  "schema": 2,
	  "modelId": "model/framework/laptop-13-amd-ryzen-7040",
	  "sourceEntity": "vendor/framework",
	  "firmwareName": "Framework UEFI",
	  "setupModeStrategy": "clear-platform-key",
	  "enrollmentBackend": "sbctl",
	  "requiredPresent": ["KEK", "db", "dbx"],
	  "requiredAbsent": ["PK"],
	  "preserveFirmwareBuiltin": ["KEK", "db"],
	  "untouched": ["dbx"],
	  "factoryOwnershipProof": "pk-equals-pkdefault",
	  "instructions": ["Delete only PK."],
	  "command": "rm -rf /"
	}`

	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFirmwarePolicySnapshot(path)
	if err == nil {
		t.Fatal("accepted unknown privileged policy field")
	}
}

func TestFirmwarePolicySnapshotRejectsArbitraryEFIName(t *testing.T) {
	snapshot, err := SnapshotFirmwarePolicy(
		"model/framework/laptop-13-amd-ryzen-7040",
		frameworkEffectivePolicy(),
	)
	if err != nil {
		t.Fatal(err)
	}

	snapshot.RequiredPresent = append(
		snapshot.RequiredPresent,
		"DefinitelyNotEFI",
	)

	err = ValidateFirmwarePolicySnapshot(snapshot)
	if err == nil ||
		!strings.Contains(err.Error(), "unsupported EFI variable") {
		t.Fatalf("got %v", err)
	}
}

func TestFirmwareInstructionsReturnsValidatedSnapshotInstructions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "firmware-policy.json")

	snapshot := enrollmentTestSnapshot()
	data, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}

	instructions, err := FirmwareInstructions(path)
	if err != nil {
		t.Fatal(err)
	}

	want := snapshot.Instructions
	if !reflect.DeepEqual(instructions, want) {
		t.Fatalf("instructions = %#v, want %#v", instructions, want)
	}
}

func TestFirmwareInstructionsRejectsUnvalidatedSnapshot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "firmware-policy.json")

	snapshot := enrollmentTestSnapshot()
	snapshot.SetupModeStrategy = "run-whatever"

	data, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := FirmwareInstructions(path); err == nil {
		t.Fatal("accepted instructions from invalid firmware policy snapshot")
	}
}
