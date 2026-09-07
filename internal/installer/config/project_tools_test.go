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
