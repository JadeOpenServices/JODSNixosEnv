package oddc

import "errors"

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
	return source.resolveCanonical(identity)
}
