package oddc

import (
	"errors"

	portable "github.com/bakanura/gjallarOS/pkg/oddc"
)

var (
	ErrNoMatch        = errors.New("no canonical ODDC model matched")
	ErrAmbiguousMatch = errors.New("ambiguous canonical ODDC model match")
)

type SourceMetadata struct {
	Kind       string
	Repository string
	Revision   string
	Integrity  string
}

type Resolved struct {
	ModelID     string
	Canonical   CanonicalResolution
	Validations []Validation
	Source      SourceMetadata
}

type DeviceSource interface {
	Resolve(identity Identity) (Resolved, error)
	Metadata() SourceMetadata
}

type HostOverlay = portable.Overlay

// HostOverlayDeviceSource is an optional capability for sources that can
// resolve immutable canonical ODDC data together with separate machine-local
// host state. Host overlays are intentionally not source metadata.
type HostOverlayDeviceSource interface {
	DeviceSource
	ResolveWithHost(
		identity Identity,
		host []HostOverlay,
	) (Resolved, error)
}

type EmbeddedSource struct {
	Root       string
	Repository string
	Revision   string
	Integrity  string
}

func (source EmbeddedSource) Metadata() SourceMetadata {
	return SourceMetadata{
		Kind:       "embedded",
		Repository: source.Repository,
		Revision:   source.Revision,
		Integrity:  source.Integrity,
	}
}

func (source EmbeddedSource) Resolve(
	identity Identity,
) (Resolved, error) {
	return source.resolveCanonical(identity, nil)
}

func (source EmbeddedSource) ResolveWithHost(
	identity Identity,
	host []HostOverlay,
) (Resolved, error) {
	return source.resolveCanonical(identity, host)
}
