package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizeWebApplicationsExplicitEmptyOverridesLegacyState(t *testing.T) {
	user := User{
		WebApplications:  []WebApplicationIntent{},
		PlaneEnable:      true,
		PlaneHost:        "https://stale-plane.example.test",
		DrawioEnable:     true,
		DrawioSelfHosted: true,
		DrawioHost:       "https://stale-drawio.example.test",
	}

	if err := NormalizeWebApplications(&user); err != nil {
		t.Fatal(err)
	}

	if user.WebApplications == nil {
		t.Fatal("explicit canonical empty list became absent")
	}

	if len(user.WebApplications) != 0 {
		t.Fatalf(
			"webApplications = %#v, want explicit empty list",
			user.WebApplications,
		)
	}

	if user.PlaneEnable ||
		user.PlaneHost != "" ||
		user.DrawioEnable ||
		user.DrawioSelfHosted ||
		user.DrawioHost != "" {
		t.Fatalf(
			"stale legacy state survived canonical empty intent: %+v",
			user,
		)
	}
}

func TestNormalizeWebApplicationsAbsentFieldStillMigratesLegacyState(t *testing.T) {
	user := User{
		PlaneEnable: true,
		PlaneHost:   "plane.example.test",
	}

	if user.WebApplications != nil {
		t.Fatal("test requires absent webApplications state")
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
	}

	if len(user.WebApplications) != len(want) {
		t.Fatalf(
			"legacy migration produced %#v, want %#v",
			user.WebApplications,
			want,
		)
	}

	for index := range want {
		if user.WebApplications[index] != want[index] {
			t.Fatalf(
				"legacy migration[%d] = %#v, want %#v",
				index,
				user.WebApplications[index],
				want[index],
			)
		}
	}
}

func TestWriteAtomicPersistsCanonicalEmptyWebApplications(t *testing.T) {
	path := filepath.Join(
		t.TempDir(),
		"user.config.json",
	)

	user := User{
		WebApplications: []WebApplicationIntent{},
	}

	if err := WriteAtomic(path, user); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	text := string(data)

	if !strings.Contains(
		text,
		`"webApplications": []`,
	) {
		t.Fatalf(
			"canonical empty webApplications was omitted:\n%s",
			text,
		)
	}

	if strings.Contains(
		text,
		`"webApplications": null`,
	) {
		t.Fatalf(
			"canonical empty webApplications persisted as null:\n%s",
			text,
		)
	}
}
