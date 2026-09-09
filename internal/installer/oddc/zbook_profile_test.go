package oddc

import (
	"path/filepath"
	"testing"
)

func TestEmbeddedRepositoryResolvesHPZBookX2G4(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "oddc"))
	if err != nil {
		t.Fatal(err)
	}

	resolved, err := (EmbeddedSource{
		Root:       root,
		Repository: "embedded:oddc",
	}).Resolve(Identity{
		FormFactor:  "laptop",
		SysVendor:   "HP",
		ProductName: "HP ZBook x2 G4",
		BoardVendor: "HP",
		BoardName:   "824C",
	})
	if err != nil {
		t.Fatal(err)
	}

	if resolved.Device.ID != "laptop/hp/zbook-x2-g4" {
		t.Fatalf("resolved device=%q", resolved.Device.ID)
	}

	want := []string{
		"laptop/common",
		"laptop/hp",
		"laptop/hp/zbook-x2-g4",
	}

	if len(resolved.Inheritance) != len(want) {
		t.Fatalf(
			"inheritance=%v want=%v",
			resolved.Inheritance,
			want,
		)
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

func TestEmbeddedRepositoryZBookMatchSurvivesFirmwareRevisionChanges(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", "..", "oddc"))
	if err != nil {
		t.Fatal(err)
	}

	resolved, err := (EmbeddedSource{
		Root: root,
	}).Resolve(Identity{
		FormFactor:     "laptop",
		SysVendor:      "HP",
		ProductName:    "HP ZBook x2 G4",
		ProductVersion: "different-sku-revision",
		BoardVendor:    "HP",
		BoardName:      "824C",
		BoardVersion:   "different-firmware-revision",
	})
	if err != nil {
		t.Fatal(err)
	}

	if resolved.Device.ID != "laptop/hp/zbook-x2-g4" {
		t.Fatalf(
			"firmware revision changed resolution to %q",
			resolved.Device.ID,
		)
	}
}
