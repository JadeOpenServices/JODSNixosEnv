package config

import "testing"

func TestValidateUpdateChannel(t *testing.T) {
	user := User{
		Hostname:    "gjallarOS",
		Username:    "baka",
		Theme:       "noctalia",
		Shell:       "zsh",
		Editors:     []string{"vscodium"},
		Browsers:    []string{"librewolf"},
		AIAgentMode: "workspace",
	}
	for _, channel := range []string{"", "stable", "main"} {
		user.UpdateChannel = channel
		if err := Validate(user); err != nil {
			t.Fatalf("channel %q rejected: %v", channel, err)
		}
	}
	for _, channel := range []string{"Stable", "unstable", "v26.05.1"} {
		user.UpdateChannel = channel
		if err := Validate(user); err == nil {
			t.Fatalf("channel %q accepted", channel)
		}
	}
}
