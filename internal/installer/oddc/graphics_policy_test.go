package oddc

import "testing"

func TestHPZBookPinsLegacy580(t *testing.T) {
	resolved, err := repositoryODDCSource(t).Resolve(Identity{
		FormFactor:  "laptop",
		SysVendor:   "HP",
		ProductName: "HP ZBook x2 G4",
		BoardName:   "824C",
	})
	if err != nil {
		t.Fatal(err)
	}

	effective, err := ResolveGraphicsPolicy(resolved)
	if err != nil {
		t.Fatal(err)
	}

	if effective.Policy.DriverBranch != "legacy_580" {
		t.Fatalf("DriverBranch=%q", effective.Policy.DriverBranch)
	}
	if effective.Policy.IntegratedKernelDriver != "i915" {
		t.Fatalf("IntegratedKernelDriver=%q", effective.Policy.IntegratedKernelDriver)
	}
	if effective.Policy.DiscreteKernelDriver != "nvidia" {
		t.Fatalf("DiscreteKernelDriver=%q", effective.Policy.DiscreteKernelDriver)
	}
	if effective.SourceEntity != "model/hp/zbook-x2-g4" {
		t.Fatalf("SourceEntity=%q", effective.SourceEntity)
	}
}
