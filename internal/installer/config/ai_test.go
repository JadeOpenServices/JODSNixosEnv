package config

import "testing"

func validAIUser() User {
	return User{System: "x86_64-linux", Profile: "laptop", Hostname: "host", Username: "user", Theme: "nord", Shell: "zsh", Editors: []string{"vim"}, Browsers: []string{"firefox"}, AIEnable: true, AIAgentMode: "workspace"}
}

func TestAIConfigurationBranches(t *testing.T) {
	u := validAIUser()
	u.AIEnable = false
	u.AIAgentMode = ""
	if err := Validate(u); err != nil {
		t.Fatalf("disabled: %v", err)
	}
	u = validAIUser()
	if err := Validate(u); err != nil {
		t.Fatalf("automatic: %v", err)
	}
	u.OverrideAISelection = true
	u.OverrideModelWith = "qwen2.5-coder:7b"
	if err := Validate(u); err != nil {
		t.Fatalf("override: %v", err)
	}
	u.OverrideModelWith = ""
	if err := Validate(u); err == nil {
		t.Fatal("invalid override accepted")
	}
}
