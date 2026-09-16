package deviceprofile

import (
	"testing"

	"github.com/bakanura/gjallarOS/internal/installer/discovery"
	"github.com/bakanura/gjallarOS/internal/installer/oddc"
)

type unmatchedCanonicalSource struct{}

func (unmatchedCanonicalSource) Resolve(
	oddc.Identity,
) (oddc.Resolved, error) {
	return oddc.Resolved{}, oddc.ErrNoMatch
}

func (unmatchedCanonicalSource) Metadata() oddc.SourceMetadata {
	return oddc.SourceMetadata{
		Kind:     "embedded",
		Revision: "test",
	}
}

func TestResolveAcceptsUnmatchedLaptopWithoutInventingModel(
	t *testing.T,
) {
	resolved, err := Resolve(
		unmatchedCanonicalSource{},
		discovery.Hardware{
			FormFactor:  "laptop",
			SysVendor:   "Unknown",
			ProductName: "Unknown Laptop",
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if resolved.ModelID != "" {
		t.Fatalf(
			"ModelID=%q, want unmatched",
			resolved.ModelID,
		)
	}

	if resolved.Source.Revision != "test" {
		t.Fatalf(
			"Source=%+v",
			resolved.Source,
		)
	}
}
