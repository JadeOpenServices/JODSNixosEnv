package oddc

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
)

const ManifestSchema = 1

var idPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*(/[a-z0-9][a-z0-9._-]*)+$`)

type Identity struct {
	FormFactor     string
	SysVendor      string
	ProductName    string
	ProductVersion string
	BoardVendor    string
	BoardName      string
	BoardVersion   string
}

type Match struct {
	SysVendor      []string `json:"sysVendor,omitempty"`
	ProductName    []string `json:"productName,omitempty"`
	ProductVersion []string `json:"productVersion,omitempty"`
	BoardVendor    []string `json:"boardVendor,omitempty"`
	BoardName      []string `json:"boardName,omitempty"`
	BoardVersion   []string `json:"boardVersion,omitempty"`
}

type Lifecycle struct {
	Status string `json:"status"`
}

type Validation struct {
	LastValidatedNixOS             string `json:"lastValidatedNixOS,omitempty"`
	LastValidatedGjallarOSRevision string `json:"lastValidatedGjallarOSRevision,omitempty"`
	LastValidatedAt                string `json:"lastValidatedAt,omitempty"`
}

type Manifest struct {
	Schema     int        `json:"schema"`
	ID         string     `json:"id"`
	Class      string     `json:"class"`
	Vendor     string     `json:"vendor,omitempty"`
	Match      Match      `json:"match,omitempty"`
	Inherits   []string   `json:"inherits,omitempty"`
	Modules    []string   `json:"modules,omitempty"`
	Lifecycle  Lifecycle  `json:"lifecycle"`
	Validation Validation `json:"validation"`
}

func DecodeManifest(r io.Reader) (Manifest, error) {
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()

	var manifest Manifest
	if err := decoder.Decode(&manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode oddc manifest: %w", err)
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return Manifest{}, fmt.Errorf("decode oddc manifest: trailing JSON value")
		}
		return Manifest{}, fmt.Errorf("decode oddc manifest trailing data: %w", err)
	}

	if err := ValidateManifest(manifest); err != nil {
		return Manifest{}, err
	}

	return manifest, nil
}

func ValidateManifest(manifest Manifest) error {
	if manifest.Schema != ManifestSchema {
		return fmt.Errorf(
			"unsupported oddc manifest schema %d for %q",
			manifest.Schema,
			manifest.ID,
		)
	}

	if !validID(manifest.ID) {
		return fmt.Errorf("invalid oddc device id %q", manifest.ID)
	}

	if strings.TrimSpace(manifest.Class) == "" {
		return fmt.Errorf("oddc manifest %q has no device class", manifest.ID)
	}

	switch manifest.Lifecycle.Status {
	case "supported", "experimental", "stale", "retired":
	default:
		return fmt.Errorf(
			"oddc manifest %q has invalid lifecycle status %q",
			manifest.ID,
			manifest.Lifecycle.Status,
		)
	}

	for _, inherited := range manifest.Inherits {
		if !validID(inherited) {
			return fmt.Errorf(
				"oddc manifest %q has invalid inherited id %q",
				manifest.ID,
				inherited,
			)
		}
	}

	for _, module := range manifest.Modules {
		if err := validateRelativePath(module); err != nil {
			return fmt.Errorf(
				"oddc manifest %q module %q: %w",
				manifest.ID,
				module,
				err,
			)
		}
	}

	return nil
}

func validID(value string) bool {
	return idPattern.MatchString(strings.TrimSpace(value))
}

func validateRelativePath(value string) error {
	if value == "" {
		return fmt.Errorf("path is empty")
	}

	if filepath.IsAbs(value) {
		return fmt.Errorf("absolute path is not allowed")
	}

	clean := filepath.Clean(value)
	if clean == "." ||
		clean == ".." ||
		strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path escapes device directory")
	}

	if filepath.ToSlash(clean) != filepath.ToSlash(value) {
		return fmt.Errorf("path is not canonical")
	}

	return nil
}
