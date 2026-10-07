package config

import "testing"

func validAIUser() User {
	return User{
		Hostname:    "host",
		Username:    "user",
		Theme:       "nord",
		Shell:       "zsh",
		Editors:     []string{"vim"},
		Browsers:    []string{"firefox"},
		AIEnable:    true,
		AIAgentMode: "workspace",
	}
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

func TestAIEndpointValidation(t *testing.T) {
	u := validAIUser()
	u.AIEndpoint = "https://192.168.8.205"
	u.AIRemoteModel = "qwen3-coder:30b"
	u.AIRemoteContextTokens = 32768
	if err := Validate(u); err != nil {
		t.Fatalf("central server: %v", err)
	}

	for name, mutate := range map[string]func(*User){
		"scheme":    func(u *User) { u.AIEndpoint = "ftp://192.168.8.205:11434" },
		"http":      func(u *User) { u.AIEndpoint = "http://192.168.8.205:11434" },
		"no host":   func(u *User) { u.AIEndpoint = "https://" },
		"path":      func(u *User) { u.AIEndpoint = "https://192.168.8.205/v1" },
		"userinfo":  func(u *User) { u.AIEndpoint = "https://a:b@192.168.8.205" },
		"no model":  func(u *User) { u.AIRemoteModel = "" },
		"bad model": func(u *User) { u.AIRemoteModel = "qwen3 coder" },
		"flag":      func(u *User) { u.AIRemoteModel = "-x" },
		"tokens":    func(u *User) { u.AIRemoteContextTokens = 0 },
	} {
		bad := u
		mutate(&bad)
		if err := Validate(bad); err == nil {
			t.Fatalf("%s: invalid central server accepted", name)
		}
	}
}
