package config

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	WebApplicationPlane  = "plane"
	WebApplicationDrawio = "drawio"
	WebApplicationTeams  = "teams"
)

// WebApplicationIntent describes which web application the user wants
// integrated and, when applicable, which endpoint that integration should use.
//
// Browser choice, profile layout, MIME routing and application-specific launch
// behavior remain owned by the corresponding application module.
type WebApplicationIntent struct {
	ID       string `json:"id"`
	Endpoint string `json:"endpoint,omitempty"`
}

// MarshalJSON writes only the canonical web-application contract.
//
// PlaneEnable, PlaneHost, DrawioEnable, DrawioSelfHosted and DrawioHost remain
// on User solely so pre-GJAL-95 JSON can still be decoded and normalized. The
// shadow fields below suppress those migration-only values from every new
// user.config.json write.
func (user User) MarshalJSON() ([]byte, error) {
	type CanonicalUser User

	return json.Marshal(struct {
		CanonicalUser

		PlaneEnable      *bool   `json:"planeEnable,omitempty"`
		PlaneHost        *string `json:"planeHost,omitempty"`
		DrawioEnable     *bool   `json:"drawioEnable,omitempty"`
		DrawioSelfHosted *bool   `json:"drawioSelfHosted,omitempty"`
		DrawioHost       *string `json:"drawioHost,omitempty"`
	}{
		CanonicalUser: CanonicalUser(user),
	})
}

// SupportedWebApplicationIDs returns the stable integration identifiers exposed
// by the installer contract.
func SupportedWebApplicationIDs() []string {
	return []string{
		WebApplicationPlane,
		WebApplicationDrawio,
		WebApplicationTeams,
	}
}

func supportedWebApplication(id string) bool {
	for _, candidate := range SupportedWebApplicationIDs() {
		if id == candidate {
			return true
		}
	}
	return false
}

// NormalizeWebApplications canonicalizes the generic integration contract.
//
// During the migration, legacy Plane/Draw.io fields remain as a compatibility
// bridge for configurations that predate webApplications. Generic intent is
// canonical whenever the webApplications field is present, including an empty list.
func NormalizeWebApplications(user *User) error {
	if user.WebApplications == nil {
		return webApplicationsFromLegacy(user)
	}

	normalized := make([]WebApplicationIntent, 0, len(user.WebApplications))
	seen := make(map[string]bool, len(user.WebApplications))

	for index, application := range user.WebApplications {
		id := strings.ToLower(strings.TrimSpace(application.ID))
		if !supportedWebApplication(id) {
			return fmt.Errorf(
				"webApplications[%d].id: unsupported web application %q",
				index,
				application.ID,
			)
		}
		if seen[id] {
			return fmt.Errorf(
				"webApplications[%d].id: duplicate web application %q",
				index,
				id,
			)
		}
		seen[id] = true

		endpoint := strings.TrimSpace(application.Endpoint)
		if endpoint != "" {
			value, err := NormalizeExternalServiceEndpoint(endpoint)
			if err != nil {
				return fmt.Errorf(
					"webApplications[%d].endpoint: %w",
					index,
					err,
				)
			}
			endpoint = value
		}

		if id == WebApplicationPlane && endpoint == "" {
			return fmt.Errorf(
				"webApplications[%d].endpoint: Plane requires an endpoint",
				index,
			)
		}

		normalized = append(normalized, WebApplicationIntent{
			ID:       id,
			Endpoint: endpoint,
		})
	}

	user.WebApplications = normalized
	applyLegacyWebApplicationCompatibility(user)

	return nil
}

func webApplicationsFromLegacy(user *User) error {
	// Teams was an unconditional desktop integration before webApplications
	// became canonical. Preserve that behavior only while migrating a config
	// where the canonical field is genuinely absent.
	applications := []WebApplicationIntent{
		{ID: WebApplicationTeams},
	}

	if user.PlaneEnable {
		endpoint, err := NormalizeExternalServiceEndpoint(user.PlaneHost)
		if err != nil {
			return fmt.Errorf("planeHost: %w", err)
		}

		user.PlaneHost = endpoint
		applications = append(applications, WebApplicationIntent{
			ID:       WebApplicationPlane,
			Endpoint: endpoint,
		})
	}

	if user.DrawioEnable {
		endpoint := ""

		if user.DrawioSelfHosted {
			value, err := NormalizeExternalServiceEndpoint(user.DrawioHost)
			if err != nil {
				return fmt.Errorf("drawioHost: %w", err)
			}

			endpoint = value
			user.DrawioHost = value
		} else {
			// Public diagrams.net mode does not require a custom endpoint.
			user.DrawioHost = ""
		}

		applications = append(applications, WebApplicationIntent{
			ID:       WebApplicationDrawio,
			Endpoint: endpoint,
		})
	}

	user.WebApplications = applications
	return nil
}

func applyLegacyWebApplicationCompatibility(user *User) {
	user.PlaneEnable = false
	user.PlaneHost = ""
	user.DrawioEnable = false
	user.DrawioSelfHosted = false
	user.DrawioHost = ""

	for _, application := range user.WebApplications {
		switch application.ID {
		case WebApplicationPlane:
			user.PlaneEnable = true
			user.PlaneHost = application.Endpoint

		case WebApplicationDrawio:
			user.DrawioEnable = true
			if application.Endpoint != "" {
				user.DrawioSelfHosted = true
				user.DrawioHost = application.Endpoint
			}
		}
	}
}
