package deviceprofile

import (
	"testing"

	"github.com/bakanura/gjallarOS/internal/installer/discovery"
	"github.com/bakanura/gjallarOS/internal/installer/oddc"
)

type noMatchSource struct{}

func (noMatchSource) Resolve(oddc.Identity) (oddc.Resolved, error) {
	return oddc.Resolved{}, oddc.ErrNoMatch
}

func (noMatchSource) Materialize(
	oddc.Resolved,
	string,
) (oddc.Materialized, error) {
	return oddc.Materialized{}, nil
}

func (noMatchSource) Metadata() oddc.SourceMetadata {
	return oddc.SourceMetadata{
		Kind:     "test",
		Revision: "test-revision",
	}
}

func TestResolvePreservesSourceForUnmatchedDesktop(t *testing.T) {
	resolved, err := Resolve(
		noMatchSource{},
		discovery.Hardware{FormFactor: "desktop"},
	)
	if err != nil {
		t.Fatal(err)
	}

	if resolved.Device.ID != "" {
		t.Fatalf("unexpected device %q", resolved.Device.ID)
	}

	if resolved.Source.Revision != "test-revision" {
		t.Fatalf(
			"Source.Revision=%q",
			resolved.Source.Revision,
		)
	}
}

func TestResolveRequiresLaptopFallback(t *testing.T) {
	_, err := Resolve(
		noMatchSource{},
		discovery.Hardware{FormFactor: "laptop"},
	)
	if err == nil {
		t.Fatal("unmatched laptop was accepted")
	}
}
