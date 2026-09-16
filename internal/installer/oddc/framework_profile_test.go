package oddc

import (
	"errors"
	"testing"
)

func TestEmbeddedRepositoryResolvesFramework13AMD7040Exactly(
	t *testing.T,
) {
	resolved, err := repositoryODDCSource(t).Resolve(
		Identity{
			FormFactor:  "laptop",
			SysVendor:   "Framework",
			ProductName: "Laptop 13 (AMD Ryzen 7040Series)",
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	const want = "model/framework/laptop-13-amd-ryzen-7040"

	if resolved.ModelID != want {
		t.Fatalf(
			"ModelID=%q want=%q",
			resolved.ModelID,
			want,
		)
	}
}

func TestEmbeddedRepositoryUnknownFrameworkHasNoCanonicalModel(
	t *testing.T,
) {
	_, err := repositoryODDCSource(t).Resolve(
		Identity{
			FormFactor:  "laptop",
			SysVendor:   "Framework",
			ProductName: "Unknown Framework Laptop",
		},
	)

	if !errors.Is(err, ErrNoMatch) {
		t.Fatalf(
			"expected ErrNoMatch, got %v",
			err,
		)
	}
}
