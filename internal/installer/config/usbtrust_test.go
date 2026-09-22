package config

import "testing"

func TestValidateUSBTrustIntent(t *testing.T) {
	base := User{
		Hostname:          "gjallarOS",
		Username:          "baka",
		Theme:             "noctalia",
		Shell:             "zsh",
		Editors:           []string{"vscodium"},
		Browsers:          []string{"librewolf"},
		AIAgentMode:       "workspace",
		USBGuardEnable:    true,
		USBTrustTPMHandle: "0x81000042",
	}

	if err := Validate(base); err != nil {
		t.Fatalf("valid USB trust intent rejected: %v", err)
	}

	tests := []struct {
		name string
		edit func(*User)
	}{
		{
			name: "invalid handle",
			edit: func(u *User) { u.USBTrustTPMHandle = "0x8100000" },
		},
		{
			name: "enforcement without backend",
			edit: func(u *User) {
				u.USBGuardEnable = false
				u.USBTrustEnforce = true
			},
		},
		{
			name: "enforcement without signer",
			edit: func(u *User) {
				u.USBTrustEnforce = true
				u.USBTrustTPMHandle = ""
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := base
			test.edit(&got)
			if err := Validate(got); err == nil {
				t.Fatal("invalid USB trust intent accepted")
			}
		})
	}
}
