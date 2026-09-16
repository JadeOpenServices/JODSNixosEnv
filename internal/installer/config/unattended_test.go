package config

import (
	"encoding/json"
	"testing"
)

func TestUnattendedInstallDefaultsFalse(t *testing.T) {
	var user User

	if err := json.Unmarshal([]byte(`{
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

func TestForceRedeployDefaultsFalse(t *testing.T) {
	var user User

	if err := json.Unmarshal([]byte(`{
		"profile": "laptop"
	}`), &user); err != nil {
		t.Fatal(err)
	}

	if user.ForceRedeploy {
		t.Fatal("force redeployment enabled without explicit configuration")
	}
}

func TestForceRedeployCanBeExplicitlyEnabled(t *testing.T) {
	var user User

	if err := json.Unmarshal([]byte(`{
		"forceRedeploy": true
	}`), &user); err != nil {
		t.Fatal(err)
	}

	if !user.ForceRedeploy {
		t.Fatal("explicit forceRedeploy=true was not decoded")
	}
}
