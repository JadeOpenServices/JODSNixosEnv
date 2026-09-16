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

	if effective.SourceEntity != "vendor/framework" {
		t.Fatalf("SourceEntity=%q", effective.SourceEntity)
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

func TestHPZBookWithoutSecureBootPolicyIsUnsupported(t *testing.T) {
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

	if effective.SourceEntity != "" {
		t.Fatalf(
			"unsupported policy unexpectedly has source entity %q",
			effective.SourceEntity,
		)
	}

	if effective.Policy.SetupModeStrategy != "unsupported" {
		t.Fatalf(
			"SetupModeStrategy=%q",
			effective.Policy.SetupModeStrategy,
		)
	}

	if strings.TrimSpace(effective.Policy.UnsupportedReason) == "" {
		t.Fatal(
			"absent canonical Secure Boot policy must fail closed with an explanation",
		)
	}
}
