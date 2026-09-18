package config

import (
	"strings"
	"testing"
)

func TestNormalizeProjectToolsNextcloud(t *testing.T) {
	user := User{
		NextcloudEnable:    true,
		NextcloudHost:      "https://cloud.example.test///",
		NextcloudLocalRoot: "/home/test/Nextcloud",
	}

	if err := NormalizeProjectTools(&user); err != nil {
		t.Fatal(err)
	}

	if user.NextcloudHost != "https://cloud.example.test" {
		t.Fatalf(
			"NextcloudHost = %q",
			user.NextcloudHost,
		)
	}

	if user.NextcloudLocalRoot != "/home/test/Nextcloud" {
		t.Fatalf(
			"NextcloudLocalRoot = %q",
			user.NextcloudLocalRoot,
		)
	}
}

func TestNormalizeProjectToolsNextcloudRequiresHTTPS(
	t *testing.T,
) {
	user := User{
		NextcloudEnable:    true,
		NextcloudHost:      "http://cloud.example.test",
		NextcloudLocalRoot: "/home/test/Nextcloud",
	}

	err := NormalizeProjectTools(&user)
	if err == nil ||
		!strings.Contains(err.Error(), "HTTPS is required") {
		t.Fatalf(
			"error = %v, want HTTPS requirement",
			err,
		)
	}
}

func TestNormalizeProjectToolsNextcloudRequiresAbsoluteLocalRoot(
	t *testing.T,
) {
	user := User{
		NextcloudEnable:    true,
		NextcloudHost:      "https://cloud.example.test",
		NextcloudLocalRoot: "Nextcloud",
	}

	err := NormalizeProjectTools(&user)
	if err == nil ||
		!strings.Contains(err.Error(), "absolute path required") {
		t.Fatalf(
			"error = %v, want absolute-path requirement",
			err,
		)
	}
}

func TestNormalizeProjectToolsDisabledNextcloudClearsState(
	t *testing.T,
) {
	user := User{
		NextcloudEnable:    false,
		NextcloudHost:      "https://stale.example.test",
		NextcloudLocalRoot: "/stale",
	}

	if err := NormalizeProjectTools(&user); err != nil {
		t.Fatal(err)
	}

	if user.NextcloudHost != "" ||
		user.NextcloudLocalRoot != "" {
		t.Fatalf(
			"disabled Nextcloud retained state: host=%q root=%q",
			user.NextcloudHost,
			user.NextcloudLocalRoot,
		)
	}
}
