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

	if resolved.Source.Revision != "test-revision" {
		t.Fatalf(
			"Source.Revision=%q",
			resolved.Source.Revision,
		)
	}
}
