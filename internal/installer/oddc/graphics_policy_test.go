package oddc

import "testing"

func TestGraphicsPolicyUsesMostSpecificOverride(t *testing.T) {
	parent := Manifest{
		ID:       "laptop/vendor",
		Graphics: &GraphicsPolicy{DriverBranch: "stable"},
	}
	child := Manifest{
		ID:       "laptop/vendor/model",
		Graphics: &GraphicsPolicy{DriverBranch: "legacy_580"},
	}

	effective, err := ResolveGraphicsPolicy(Resolved{
		Device:      child,
		Inheritance: []Manifest{parent, child},
	})
	if err != nil {
		t.Fatal(err)
	}

	if effective.Policy.DriverBranch != "legacy_580" {
		t.Fatalf("DriverBranch=%q", effective.Policy.DriverBranch)
	}
	if effective.SourceLayer != child.ID {
		t.Fatalf("SourceLayer=%q", effective.SourceLayer)
	}
}

func TestMissingGraphicsPolicyLeavesDynamicDetectionUnchanged(t *testing.T) {
	effective, err := ResolveGraphicsPolicy(Resolved{
		Device: Manifest{ID: "laptop/common"},
		Inheritance: []Manifest{
			{ID: "laptop/common"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if effective.Policy.DriverBranch != "" {
		t.Fatalf("DriverBranch=%q, want empty", effective.Policy.DriverBranch)
	}
}

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
	if effective.SourceLayer != "laptop/hp/zbook-x2-g4" {
		t.Fatalf("SourceLayer=%q", effective.SourceLayer)
	}
}
