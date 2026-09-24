package config

import (
	"strings"
	"testing"
)

func TestSupportedWebApplicationIDs(t *testing.T) {
	got := SupportedWebApplicationIDs()
	want := []string{"plane", "drawio", "teams"}

	if len(got) != len(want) {
		t.Fatalf("SupportedWebApplicationIDs() = %#v, want %#v", got, want)
	}

	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("SupportedWebApplicationIDs() = %#v, want %#v", got, want)
		}
	}
}

func TestNormalizeWebApplicationsCanonicalizesGenericIntent(t *testing.T) {
	user := User{
		WebApplications: []WebApplicationIntent{
			{
				ID:       " Plane ",
				Endpoint: "plane.example.test///",
			},
			{
				ID: "DRAWIO",
			},
			{
				ID: " teams ",
			},
		},
	}

	if err := NormalizeWebApplications(&user); err != nil {
		t.Fatal(err)
	}

	want := []WebApplicationIntent{
		{
			ID:       WebApplicationPlane,
			Endpoint: "http://plane.example.test",
		},
		{
			ID: WebApplicationDrawio,
		},
		{
			ID: WebApplicationTeams,
		},
	}

	if len(user.WebApplications) != len(want) {
		t.Fatalf("web applications = %#v, want %#v", user.WebApplications, want)
	}

	for index := range want {
		if user.WebApplications[index] != want[index] {
			t.Fatalf(
				"webApplications[%d] = %#v, want %#v",
				index,
				user.WebApplications[index],
				want[index],
			)
		}
	}

	if !user.PlaneEnable || user.PlaneHost != "http://plane.example.test" {
		t.Fatalf("Plane compatibility bridge incorrect: %#v", user)
	}

	if !user.DrawioEnable || user.DrawioSelfHosted || user.DrawioHost != "" {
		t.Fatalf("Draw.io public compatibility bridge incorrect: %#v", user)
	}
}

func TestNormalizeWebApplicationsMapsSelfHostedDrawioCompatibility(t *testing.T) {
	user := User{
		WebApplications: []WebApplicationIntent{
			{
				ID:       WebApplicationDrawio,
				Endpoint: "https://drawio.example.test///",
			},
		},
	}

	if err := NormalizeWebApplications(&user); err != nil {
		t.Fatal(err)
	}

	if !user.DrawioEnable ||
		!user.DrawioSelfHosted ||
		user.DrawioHost != "https://drawio.example.test" {
		t.Fatalf("self-hosted Draw.io compatibility bridge incorrect: %#v", user)
	}
}

func TestNormalizeWebApplicationsSynthesizesLegacyIntent(t *testing.T) {
	user := User{
		PlaneEnable:      true,
		PlaneHost:        "plane.example.test/",
		DrawioEnable:     true,
		DrawioSelfHosted: false,
		DrawioHost:       "ignored.example.test",
	}

	if err := NormalizeWebApplications(&user); err != nil {
		t.Fatal(err)
	}

	want := []WebApplicationIntent{
		{
			ID: WebApplicationTeams,
		},
		{
			ID:       WebApplicationPlane,
			Endpoint: "http://plane.example.test",
		},
		{
			ID: WebApplicationDrawio,
		},
	}

	if len(user.WebApplications) != len(want) {
		t.Fatalf("legacy bridge = %#v, want %#v", user.WebApplications, want)
	}

	for index := range want {
		if user.WebApplications[index] != want[index] {
			t.Fatalf(
				"legacy bridge[%d] = %#v, want %#v",
				index,
				user.WebApplications[index],
				want[index],
			)
		}
	}

	if user.DrawioHost != "" {
		t.Fatalf("public Draw.io retained unused legacy host %q", user.DrawioHost)
	}
}

func TestNormalizeWebApplicationsGenericIntentOverridesLegacyCompatibility(t *testing.T) {
	user := User{
		WebApplications: []WebApplicationIntent{
			{ID: WebApplicationTeams},
		},
		PlaneEnable:      true,
		PlaneHost:        "https://stale-plane.example.test",
		DrawioEnable:     true,
		DrawioSelfHosted: true,
		DrawioHost:       "https://stale-drawio.example.test",
	}

	if err := NormalizeWebApplications(&user); err != nil {
		t.Fatal(err)
	}

	if user.PlaneEnable ||
		user.PlaneHost != "" ||
		user.DrawioEnable ||
		user.DrawioSelfHosted ||
		user.DrawioHost != "" {
		t.Fatalf("generic intent did not replace stale compatibility state: %#v", user)
	}
}

func TestNormalizeWebApplicationsRejectsDuplicateIDs(t *testing.T) {
	user := User{
		WebApplications: []WebApplicationIntent{
			{
				ID:       WebApplicationPlane,
				Endpoint: "https://plane.example.test",
			},
			{
				ID:       "PLANE",
				Endpoint: "https://other-plane.example.test",
			},
		},
	}

	err := NormalizeWebApplications(&user)
	if err == nil || !strings.Contains(err.Error(), "duplicate web application") {
		t.Fatalf("expected duplicate ID rejection, got %v", err)
	}
}

func TestNormalizeWebApplicationsRejectsUnknownID(t *testing.T) {
	user := User{
		WebApplications: []WebApplicationIntent{
			{ID: "unknown"},
		},
	}

	err := NormalizeWebApplications(&user)
	if err == nil || !strings.Contains(err.Error(), "unsupported web application") {
		t.Fatalf("expected unsupported ID rejection, got %v", err)
	}
}

func TestNormalizeWebApplicationsRequiresPlaneEndpoint(t *testing.T) {
	user := User{
		WebApplications: []WebApplicationIntent{
			{ID: WebApplicationPlane},
		},
	}

	err := NormalizeWebApplications(&user)
	if err == nil || !strings.Contains(err.Error(), "Plane requires an endpoint") {
		t.Fatalf("expected Plane endpoint rejection, got %v", err)
	}
}

func TestNormalizeWebApplicationsAllowsTeamsBuiltInEndpoint(t *testing.T) {
	user := User{
		WebApplications: []WebApplicationIntent{
			{ID: WebApplicationTeams},
		},
	}

	if err := NormalizeWebApplications(&user); err != nil {
		t.Fatal(err)
	}

	if len(user.WebApplications) != 1 ||
		user.WebApplications[0].ID != WebApplicationTeams ||
		user.WebApplications[0].Endpoint != "" {
		t.Fatalf("unexpected Teams intent: %#v", user.WebApplications)
	}
}
