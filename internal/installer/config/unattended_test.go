package config

import (
	"encoding/json"
	"testing"
)

func TestUnattendedInstallDefaultsFalse(t *testing.T) {
	var user User

	if err := json.Unmarshal([]byte(`{
		"system": "x86_64-linux",
		"profile": "laptop"
	}`), &user); err != nil {
		t.Fatal(err)
	}

	if user.UnattendedInstall {
		t.Fatal("unattended installation enabled without explicit configuration")
	}
}

func TestUnattendedInstallCanBeExplicitlyEnabled(t *testing.T) {
	var user User

	if err := json.Unmarshal([]byte(`{
		"unattendedInstall": true
	}`), &user); err != nil {
		t.Fatal(err)
	}

	if !user.UnattendedInstall {
		t.Fatal("explicit unattendedInstall=true was not decoded")
	}
}
