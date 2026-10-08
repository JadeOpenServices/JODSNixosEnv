package deviceprofile

import (
	"errors"
	"testing"

	"github.com/JadeOpenServices/gjallarOS/internal/installer/discovery"
	"github.com/JadeOpenServices/gjallarOS/internal/installer/oddc"
)

type hostAwareTestSource struct {
	hostCalls int
}

func (source *hostAwareTestSource) Resolve(
	oddc.Identity,
) (oddc.Resolved, error) {
	return oddc.Resolved{}, errors.New(
		"ordinary Resolve must not be used when a host overlay is supplied",
	)
}

func (source *hostAwareTestSource) ResolveWithHost(
	_ oddc.Identity,
	host []oddc.HostOverlay,
) (oddc.Resolved, error) {
	source.hostCalls++

	if len(host) != 1 {
		return oddc.Resolved{}, errors.New(
			"host overlay was not passed to source",
		)
	}

	return oddc.Resolved{
		ModelID: "model/test",
		Source:  source.Metadata(),
	}, nil
}

func (*hostAwareTestSource) Metadata() oddc.SourceMetadata {
	return oddc.SourceMetadata{
		Kind:       "embedded",
		Repository: "embedded:test",
		Revision:   "test-revision",
	}
}

func TestResolveWithHostUsesExplicitHostCapability(t *testing.T) {
	source := &hostAwareTestSource{}

	resolved, err := ResolveWithHost(
		source,
		discovery.Hardware{
			FormFactor:  "laptop",
			SysVendor:   "Test",
			ProductName: "Laptop",
		},
		[]oddc.HostOverlay{
			{
				APIVersion:  "oddc.openjade.de/v2",
				ID:          "host/test",
				Kind:        "host",
				TargetModel: "model/test",
				Overrides: map[string]any{
					"hardware": map[string]any{},
				},
			},
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	if source.hostCalls != 1 {
		t.Fatalf(
			"host-aware source calls = %d, want 1",
			source.hostCalls,
		)
	}

	if resolved.ModelID != "model/test" {
		t.Fatalf("resolved model = %q", resolved.ModelID)
	}

	if resolved.Source.Revision != "test-revision" {
		t.Fatalf(
			"source metadata changed: %+v",
			resolved.Source,
		)
	}
}

func TestResolveWithHostRejectsSourceWithoutCapability(t *testing.T) {
	_, err := ResolveWithHost(
		noMatchSource{},
		discovery.Hardware{FormFactor: "desktop"},
		[]oddc.HostOverlay{
			{
				APIVersion:  "oddc.openjade.de/v2",
				ID:          "host/test",
				Kind:        "host",
				TargetModel: "model/test",
				Overrides:   map[string]any{},
			},
		},
	)

	if err == nil {
		t.Fatal("source without host-overlay capability was accepted")
	}
}
