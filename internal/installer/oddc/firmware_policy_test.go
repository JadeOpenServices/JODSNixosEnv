package oddc

import (
	"path/filepath"
	"strings"
	"testing"
)

func repositoryODDCSource(t *testing.T) EmbeddedSource {
	t.Helper()

	root, err := filepath.Abs(filepath.Join("..", "..", "..", "oddc"))
	if err != nil {
		t.Fatal(err)
	}

	return EmbeddedSource{
		Root:       root,
		Repository: "embedded:oddc",
	}
}

func TestFrameworkResolvesSupportedSecureBootFirmwarePolicy(t *testing.T) {
	resolved, err := repositoryODDCSource(t).Resolve(Identity{
		FormFactor:  "laptop",
		SysVendor:   "Framework",
		ProductName: "Laptop 13 (AMD Ryzen 7040Series)",
		BoardVendor: "Framework",
		BoardName:   "FRANMDCP07",
	})
	if err != nil {
		t.Fatal(err)
	}

	effective, err := ResolveSecureBootFirmwarePolicy(resolved)
	if err != nil {
		t.Fatal(err)
	}

	if !effective.Policy.Supported {
		t.Fatalf("Framework policy unexpectedly unsupported: %+v", effective)
	}

	if effective.SourceLayer != "laptop/framework" {
		t.Fatalf("SourceLayer=%q", effective.SourceLayer)
	}

	if effective.Policy.SetupModeStrategy != "clear-platform-key" {
		t.Fatalf(
			"SetupModeStrategy=%q",
			effective.Policy.SetupModeStrategy,
		)
	}

	if effective.Policy.FactoryOwnershipProof != "pk-equals-pkdefault" {
		t.Fatalf(
			"FactoryOwnershipProof=%q",
			effective.Policy.FactoryOwnershipProof,
		)
	}
}

func TestHPZBookExplicitlyBlocksSecureBootFirmwareTransfer(t *testing.T) {
	resolved, err := repositoryODDCSource(t).Resolve(Identity{
		FormFactor:  "laptop",
		SysVendor:   "HP",
		ProductName: "HP ZBook x2 G4",
		BoardName:   "824C",
	})
	if err != nil {
		t.Fatal(err)
	}

	effective, err := ResolveSecureBootFirmwarePolicy(resolved)
	if err != nil {
		t.Fatal(err)
	}

	if effective.Policy.Supported {
		t.Fatalf("HP ZBook unexpectedly inherited supported firmware policy")
	}

	if effective.SourceLayer != "laptop/hp" {
		t.Fatalf("SourceLayer=%q", effective.SourceLayer)
	}

	if effective.Policy.SetupModeStrategy != "unsupported" {
		t.Fatalf(
			"SetupModeStrategy=%q",
			effective.Policy.SetupModeStrategy,
		)
	}

	if !strings.Contains(effective.Policy.UnsupportedReason, "not been validated") {
		t.Fatalf(
			"UnsupportedReason=%q",
			effective.Policy.UnsupportedReason,
		)
	}
}

func TestChildSecureBootFirmwarePolicyReplacesParentAtomically(t *testing.T) {
	parent := Manifest{
		ID: "laptop/vendor",
		SecureBootFirmware: &SecureBootFirmwarePolicy{
			Supported:               true,
			FirmwareName:            "Parent firmware",
			SetupModeStrategy:       "clear-platform-key",
			EnrollmentBackend:       "sbctl",
			RequiredPresent:         []string{"KEK"},
			PreserveFirmwareBuiltin: []string{"KEK"},
			FactoryOwnershipProof:   "pk-equals-pkdefault",
			Instructions:            []string{"Parent instruction"},
		},
	}

	child := Manifest{
		ID: "laptop/vendor/model",
		SecureBootFirmware: &SecureBootFirmwarePolicy{
			Supported:         false,
			SetupModeStrategy: "unsupported",
			UnsupportedReason: "Model transfer is intentionally blocked.",
		},
	}

	effective, err := ResolveSecureBootFirmwarePolicy(Resolved{
		Device:      child,
		Inheritance: []Manifest{parent, child},
	})
	if err != nil {
		t.Fatal(err)
	}

	if effective.Policy.Supported {
		t.Fatal("child unsupported policy did not replace supported parent")
	}

	if len(effective.Policy.PreserveFirmwareBuiltin) != 0 {
		t.Fatalf(
			"child policy partially inherited parent's operational fields: %+v",
			effective.Policy,
		)
	}

	if effective.SourceLayer != child.ID {
		t.Fatalf("SourceLayer=%q", effective.SourceLayer)
	}
}

func TestMissingSecureBootFirmwarePolicyFailsClosed(t *testing.T) {
	effective, err := ResolveSecureBootFirmwarePolicy(Resolved{
		Device: Manifest{ID: "laptop/common"},
		Inheritance: []Manifest{
			{ID: "laptop/common"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if effective.Policy.Supported {
		t.Fatal("missing policy defaulted to supported")
	}

	if effective.Policy.SetupModeStrategy != "unsupported" {
		t.Fatalf(
			"SetupModeStrategy=%q",
			effective.Policy.SetupModeStrategy,
		)
	}
}
