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
	LastValidatedDeviceID          string `json:"lastValidatedDeviceID,omitempty"`
	LastValidatedODDCRevision      string `json:"lastValidatedODDCRevision,omitempty"`
	LastValidatedAt                string `json:"lastValidatedAt,omitempty"`
}

type ValidationTarget struct {
	NixOSRelease      string
	GjallarOSRevision string
	DeviceID          string
	ODDCRevision      string
}

func (v Validation) Matches(target ValidationTarget) bool {
	return strings.TrimSpace(v.LastValidatedNixOS) == strings.TrimSpace(target.NixOSRelease) &&
		strings.TrimSpace(v.LastValidatedGjallarOSRevision) == strings.TrimSpace(target.GjallarOSRevision) &&
		strings.TrimSpace(v.LastValidatedDeviceID) == strings.TrimSpace(target.DeviceID) &&
		strings.TrimSpace(v.LastValidatedODDCRevision) == strings.TrimSpace(target.ODDCRevision) &&
		strings.TrimSpace(v.LastValidatedAt) != ""
}

type SecureBootFirmwarePolicy struct {
	Supported               bool     `json:"supported"`
	FirmwareName            string   `json:"firmwareName,omitempty"`
	SetupModeStrategy       string   `json:"setupModeStrategy,omitempty"`
	EnrollmentBackend       string   `json:"enrollmentBackend,omitempty"`
	RequiredPresent         []string `json:"requiredPresent,omitempty"`
	RequiredAbsent          []string `json:"requiredAbsent,omitempty"`
	PreserveFirmwareBuiltin []string `json:"preserveFirmwareBuiltin,omitempty"`
	Untouched               []string `json:"untouched,omitempty"`
	FactoryOwnershipProof   string   `json:"factoryOwnershipProof,omitempty"`
	Instructions            []string `json:"instructions,omitempty"`
	UnsupportedReason       string   `json:"unsupportedReason,omitempty"`
}

type Manifest struct {
	Schema             int                       `json:"schema"`
	ID                 string                    `json:"id"`
	Class              string                    `json:"class"`
	Vendor             string                    `json:"vendor,omitempty"`
	Selectable         *bool                     `json:"selectable,omitempty"`
	Match              Match                     `json:"match,omitempty"`
	Inherits           []string                  `json:"inherits,omitempty"`
	Modules            []string                  `json:"modules,omitempty"`
	Lifecycle          Lifecycle                 `json:"lifecycle"`
	Validation         Validation                `json:"validation"`
	SecureBootFirmware *SecureBootFirmwarePolicy `json:"secureBootFirmware,omitempty"`
}

func (manifest Manifest) IsSelectable() bool {
	return manifest.Selectable == nil || *manifest.Selectable
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

	if manifest.SecureBootFirmware != nil {
		if err := validateSecureBootFirmwarePolicy(*manifest.SecureBootFirmware); err != nil {
			return fmt.Errorf(
				"oddc manifest %q secureBootFirmware: %w",
				manifest.ID,
				err,
			)
		}
	}

	return nil
}

func validateSecureBootFirmwarePolicy(policy SecureBootFirmwarePolicy) error {
	if !policy.Supported {
		if policy.SetupModeStrategy != "unsupported" {
			return fmt.Errorf(
				"unsupported policy requires setupModeStrategy %q",
				"unsupported",
			)
		}

		if strings.TrimSpace(policy.UnsupportedReason) == "" {
			return fmt.Errorf("unsupported policy requires unsupportedReason")
		}

		if strings.TrimSpace(policy.FirmwareName) != "" ||
			strings.TrimSpace(policy.EnrollmentBackend) != "" ||
			len(policy.RequiredPresent) != 0 ||
			len(policy.RequiredAbsent) != 0 ||
			len(policy.PreserveFirmwareBuiltin) != 0 ||
			len(policy.Untouched) != 0 ||
			strings.TrimSpace(policy.FactoryOwnershipProof) != "" ||
			len(policy.Instructions) != 0 {
			return fmt.Errorf(
				"unsupported policy cannot contain operational Secure Boot fields",
			)
		}

		return nil
	}

	if strings.TrimSpace(policy.FirmwareName) == "" {
		return fmt.Errorf("supported policy requires firmwareName")
	}

	switch policy.SetupModeStrategy {
	case "clear-platform-key", "firmware-setup-mode":
	default:
		return fmt.Errorf(
			"invalid setupModeStrategy %q",
			policy.SetupModeStrategy,
		)
	}

	switch policy.EnrollmentBackend {
	case "sbctl":
	default:
		return fmt.Errorf(
			"invalid enrollmentBackend %q",
			policy.EnrollmentBackend,
		)
	}

	switch policy.FactoryOwnershipProof {
	case "pk-equals-pkdefault":
	default:
		return fmt.Errorf(
			"invalid factoryOwnershipProof %q",
			policy.FactoryOwnershipProof,
		)
	}

	allowedVariables := map[string]bool{
		"PK":  true,
		"KEK": true,
		"db":  true,
		"dbx": true,
	}

	seen := map[string]string{}

	sets := []struct {
		name   string
		values []string
	}{
		{"requiredPresent", policy.RequiredPresent},
		{"requiredAbsent", policy.RequiredAbsent},
		{"preserveFirmwareBuiltin", policy.PreserveFirmwareBuiltin},
		{"untouched", policy.Untouched},
	}

	for _, set := range sets {
		local := map[string]bool{}

		for _, variable := range set.values {
			if !allowedVariables[variable] {
				return fmt.Errorf(
					"%s contains invalid EFI variable %q",
					set.name,
					variable,
				)
			}

			if local[variable] {
				return fmt.Errorf(
					"%s contains duplicate EFI variable %q",
					set.name,
					variable,
				)
			}
			local[variable] = true

			if previous, ok := seen[variable]; ok &&
				((previous == "requiredPresent" && set.name == "requiredAbsent") ||
					(previous == "requiredAbsent" && set.name == "requiredPresent")) {
				return fmt.Errorf(
					"EFI variable %q cannot be both required present and required absent",
					variable,
				)
			}

			seen[variable] = set.name
		}
	}

	if len(policy.Instructions) == 0 {
		return fmt.Errorf("supported policy requires firmware instructions")
	}

	for _, instruction := range policy.Instructions {
		if strings.TrimSpace(instruction) == "" {
			return fmt.Errorf("firmware instructions cannot contain empty entries")
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
