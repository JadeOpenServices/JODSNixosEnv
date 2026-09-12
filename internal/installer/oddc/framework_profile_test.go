package oddc

import (
	"path/filepath"
	"testing"
)

func TestEmbeddedRepositoryResolvesFramework13AMD7040Exactly(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "oddc"))
	if err != nil {
		t.Fatal(err)
	}

	resolved, err := (EmbeddedSource{
		Root: root,
	}).Resolve(Identity{
		FormFactor:     "laptop",
		SysVendor:      "Framework",
		ProductName:    "Laptop 13 (AMD Ryzen 7040Series)",
		ProductVersion: "A7",
		BoardVendor:    "Framework",
		BoardName:      "FRANMDCP07",
		BoardVersion:   "A7",
	})
	if err != nil {
		t.Fatal(err)
	}

	if resolved.Device.ID != "laptop/framework/laptop-13/amd/ryzen-7040" {
		t.Fatalf("resolved device=%q", resolved.Device.ID)
	}

	want := []string{
		"laptop/common",
		"laptop/framework",
		"laptop/framework/laptop-13/amd/ryzen-7040",
	}

	if len(resolved.Inheritance) != len(want) {
		t.Fatalf("inheritance=%v", resolved.Inheritance)
	}

	for i, id := range want {
		if resolved.Inheritance[i].ID != id {
			t.Fatalf(
				"inheritance[%d]=%q want=%q",
				i,
				resolved.Inheritance[i].ID,
				id,
			)
		}
	}
}

func TestEmbeddedRepositoryUnknownFrameworkFallsBackToLaptopCommon(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "oddc"))
	if err != nil {
		t.Fatal(err)
	}

	resolved, err := (EmbeddedSource{
		Root: root,
	}).Resolve(Identity{
		FormFactor:  "laptop",
		SysVendor:   "Framework",
		ProductName: "Future Framework Laptop 99",
		BoardVendor: "Framework",
		BoardName:   "NEWBOARD",
	})
	if err != nil {
		t.Fatal(err)
	}

	if resolved.Device.ID != "laptop/common" {
		t.Fatalf(
			"unknown Framework resolved to %q, want laptop/common",
			resolved.Device.ID,
		)
	}
}
