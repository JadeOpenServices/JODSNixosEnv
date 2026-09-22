package config

import "testing"

func TestNormalizeProjectToolsNormalizesNextcloud(t *testing.T) {
	user := User{
		NextcloudEnable: true,
		NextcloudHost:   "https://cloud.example.test///",
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
}

func TestNormalizeProjectToolsRequiresNextcloudHTTPS(t *testing.T) {
	user := User{
		NextcloudEnable: true,
		NextcloudHost:   "http://cloud.example.test",
	}

	if err := NormalizeProjectTools(&user); err == nil {
		t.Fatal("expected HTTP Nextcloud endpoint rejection")
	}
}

func TestNormalizeProjectToolsClearsDisabledNextcloud(t *testing.T) {
	user := User{
		NextcloudHost: "https://stale.example.test",
	}

	if err := NormalizeProjectTools(&user); err != nil {
		t.Fatal(err)
	}

	if user.NextcloudHost != "" {
		t.Fatalf(
			"disabled Nextcloud retained host %q",
			user.NextcloudHost,
		)
	}
}
