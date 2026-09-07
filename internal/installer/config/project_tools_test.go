package config

import (
	"strings"
	"testing"
)

func TestProjectToolsDisabledAllowEmptyHosts(t *testing.T) {
	u := User{
		PlaneEnable:  false,
		PlaneHost:    "",
		DrawioEnable: false,
		DrawioHost:   "",
	}

	if err := ValidateProjectTools(u); err != nil {
		t.Fatalf("disabled project tools should allow empty hosts: %v", err)
	}
}

func TestPlaneRequiresHostWhenEnabled(t *testing.T) {
	u := User{PlaneEnable: true}

	err := ValidateProjectTools(u)
	if err == nil || !strings.Contains(err.Error(), "planeHost") {
		t.Fatalf("expected planeHost validation error, got %v", err)
	}
}

func TestDrawioRequiresHostWhenEnabled(t *testing.T) {
	u := User{DrawioEnable: true}

	err := ValidateProjectTools(u)
	if err == nil || !strings.Contains(err.Error(), "drawioHost") {
		t.Fatalf("expected drawioHost validation error, got %v", err)
	}
}

func TestProjectToolsAcceptConfiguredHosts(t *testing.T) {
	u := User{
		PlaneEnable:  true,
		PlaneHost:    "192.0.2.203",
		DrawioEnable: true,
		DrawioHost:   "192.0.2.204:8080",
	}

	if err := ValidateProjectTools(u); err != nil {
		t.Fatalf("configured project tools should validate: %v", err)
	}
}

func TestNormalizeExternalServiceEndpoint(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "IPv4",
			in:   "192.0.2.203",
			want: "http://192.0.2.203",
		},
		{
			name: "IPv4 with port",
			in:   "192.0.2.204:8080",
			want: "http://192.0.2.204:8080",
		},
		{
			name: "explicit HTTPS",
			in:   "https://plane.example.internal",
			want: "https://plane.example.internal",
		},
		{
			name: "explicit HTTP and port",
			in:   "http://drawio.example.internal:8080",
			want: "http://drawio.example.internal:8080",
		},
		{
			name: "trailing slash",
			in:   "https://plane.example.internal/",
			want: "https://plane.example.internal",
		},
		{
			name: "duplicate trailing slashes",
			in:   "https://plane.example.internal///",
			want: "https://plane.example.internal",
		},
		{
			name: "surrounding whitespace",
			in:   "  192.0.2.203:3000  ",
			want: "http://192.0.2.203:3000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeExternalServiceEndpoint(tt.in)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("NormalizeExternalServiceEndpoint(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestNormalizeExternalServiceEndpointRejectsMalformedValues(t *testing.T) {
	tests := []string{
		"",
		"   ",
		"http://",
		"https://",
		"://broken",
		"ftp://plane.example.internal",
		"http://user:password@plane.example.internal",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			if _, err := NormalizeExternalServiceEndpoint(input); err == nil {
				t.Fatalf("expected %q to be rejected", input)
			}
		})
	}
}

func TestNormalizeProjectToolsCanonicalizesEnabledEndpoints(t *testing.T) {
	u := User{
		PlaneEnable:  true,
		PlaneHost:    "192.0.2.203",
		DrawioEnable: true,
		DrawioHost:   "192.0.2.204:8080///",
	}

	if err := NormalizeProjectTools(&u); err != nil {
		t.Fatal(err)
	}

	if u.PlaneHost != "http://192.0.2.203" {
		t.Fatalf("unexpected normalized Plane endpoint: %q", u.PlaneHost)
	}

	if u.DrawioHost != "http://192.0.2.204:8080" {
		t.Fatalf("unexpected normalized Draw.io endpoint: %q", u.DrawioHost)
	}
}

func TestNormalizeProjectToolsLeavesDisabledValuesUnused(t *testing.T) {
	u := User{
		PlaneEnable:  false,
		PlaneHost:    "",
		DrawioEnable: false,
		DrawioHost:   "",
	}

	if err := NormalizeProjectTools(&u); err != nil {
		t.Fatalf("disabled project tools should not require endpoints: %v", err)
	}
}
