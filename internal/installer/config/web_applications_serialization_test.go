package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

var legacyWebApplicationJSONKeys = []string{
	"planeEnable",
	"planeHost",
	"drawioEnable",
	"drawioSelfHosted",
	"drawioHost",
}

func TestUserMarshalJSONOmitsLegacyWebApplicationFields(t *testing.T) {
	user := User{
		WebApplications: []WebApplicationIntent{
			{
				ID:       WebApplicationPlane,
				Endpoint: "https://plane.example.test",
			},
			{
				ID: WebApplicationDrawio,
			},
		},

		PlaneEnable:      true,
		PlaneHost:        "https://plane.example.test",
		DrawioEnable:     true,
		DrawioSelfHosted: false,
	}

	data, err := json.Marshal(user)
	if err != nil {
		t.Fatal(err)
	}

	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil {
		t.Fatal(err)
	}

	if _, ok := object["webApplications"]; !ok {
		t.Fatal("canonical webApplications was not serialized")
	}

	for _, key := range legacyWebApplicationJSONKeys {
		if _, ok := object[key]; ok {
			t.Fatalf(
				"migration-only field %q was serialized",
				key,
			)
		}
	}
}

func TestUserUnmarshalJSONStillAcceptsLegacyWebApplicationFields(t *testing.T) {
	data := []byte(`{
		"planeEnable": true,
		"planeHost": "plane.example.test",
		"drawioEnable": true,
		"drawioSelfHosted": true,
		"drawioHost": "drawio.example.test"
	}`)

	var user User
	if err := json.Unmarshal(data, &user); err != nil {
		t.Fatal(err)
	}

	if user.WebApplications != nil {
		t.Fatalf(
			"legacy JSON unexpectedly populated canonical state before normalization: %#v",
			user.WebApplications,
		)
	}

	if !user.PlaneEnable ||
		user.PlaneHost != "plane.example.test" ||
		!user.DrawioEnable ||
		!user.DrawioSelfHosted ||
		user.DrawioHost != "drawio.example.test" {
		t.Fatalf(
			"legacy compatibility fields were not decoded: %+v",
			user,
		)
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
			ID:       WebApplicationDrawio,
			Endpoint: "http://drawio.example.test",
		},
	}

	if len(user.WebApplications) != len(want) {
		t.Fatalf(
			"migrated webApplications = %#v, want %#v",
			user.WebApplications,
			want,
		)
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
}

func TestWriteAtomicEmitsCanonicalWebApplicationsOnly(t *testing.T) {
	path := filepath.Join(
		t.TempDir(),
		"user.config.json",
	)

	user := User{
		WebApplications: []WebApplicationIntent{
			{
				ID:       WebApplicationPlane,
				Endpoint: "https://plane.example.test",
			},
		},

		PlaneEnable: true,
		PlaneHost:   "https://plane.example.test",
	}

	if err := WriteAtomic(path, user); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil {
		t.Fatal(err)
	}

	if _, ok := object["webApplications"]; !ok {
		t.Fatal("WriteAtomic omitted canonical webApplications")
	}

	for _, key := range legacyWebApplicationJSONKeys {
		if _, ok := object[key]; ok {
			t.Fatalf(
				"WriteAtomic emitted migration-only field %q",
				key,
			)
		}
	}
}

func TestLegacyMigrationWritesCanonicalWebApplicationsOnly(t *testing.T) {
	path := filepath.Join(
		t.TempDir(),
		"user.config.json",
	)

	user := User{
		PlaneEnable: true,
		PlaneHost:   "plane.example.test",
	}

	if err := NormalizeWebApplications(&user); err != nil {
		t.Fatal(err)
	}

	if err := WriteAtomic(path, user); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var object map[string]json.RawMessage
	if err := json.Unmarshal(data, &object); err != nil {
		t.Fatal(err)
	}

	rawApplications, ok := object["webApplications"]
	if !ok {
		t.Fatal("migrated config omitted canonical webApplications")
	}

	var applications []WebApplicationIntent
	if err := json.Unmarshal(
		rawApplications,
		&applications,
	); err != nil {
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

	if len(applications) != len(want) {
		t.Fatalf(
			"written webApplications = %#v, want %#v",
			applications,
			want,
		)
	}

	for index := range want {
		if applications[index] != want[index] {
			t.Fatalf(
				"written webApplications[%d] = %#v, want %#v",
				index,
				applications[index],
				want[index],
			)
		}
	}

	for _, key := range legacyWebApplicationJSONKeys {
		if _, ok := object[key]; ok {
			t.Fatalf(
				"migrated write retained legacy JSON key %q",
				key,
			)
		}
	}
}
