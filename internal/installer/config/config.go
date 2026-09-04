// Package config defines the typed contract between installer input,
// discovery, and the generated settings.nix file.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type User struct {
	System                 string   `json:"system"`
	Profile                string   `json:"profile"`
	Hostname               string   `json:"hostname"`
	Username               string   `json:"username"`
	Timezone               string   `json:"timezone"`
	Locale                 string   `json:"locale"`
	KeyboardLayout         string   `json:"keyboardLayout"`
	KeyboardVariant        string   `json:"keyboardVariant"`
	TouchpadWorkspaceSwipe bool     `json:"touchpadWorkspaceSwipe"`
	ClamshellEnable        bool     `json:"clamshellEnable"`
	USBGuardEnable         bool     `json:"usbguardEnable"`
	Name                   string   `json:"name"`
	Email                  string   `json:"email"`
	GitHubUsername         string   `json:"githubUsername"`
	DotfilesDir            string   `json:"dotfilesDir"`
	Shell                  string   `json:"shell"`
	Editors                []string `json:"editors"`
	Browsers               []string `json:"browsers"`
	PreferredEditor        string   `json:"preferredEditor"`
	PreferredBrowser       string   `json:"preferredBrowser"`
	Theme                  string   `json:"theme"`
	BackgroundNormal       string   `json:"backgroundNormal"`
	BackgroundWork         string   `json:"backgroundWork"`
	BackgroundGaming       string   `json:"backgroundGaming"`
	WorkUserEnable         bool     `json:"workUserEnable"`
	WorkUsername           string   `json:"workUsername"`
	WorkUserPasswordFile   string   `json:"workUserPasswordFile"`
	DockerEnable           bool     `json:"dockerEnable"`
	DebugFunctions         bool     `json:"debugFunctions"`
	AIEnable               bool     `json:"aiEnable"`
	OverrideAISelection    bool     `json:"overrideAiSelection"`
	OverrideModelWith      string   `json:"overrideModelWith"`
	EnableScrobbling       bool     `json:"enableScrobbling"`
	EnableLastfm           bool     `json:"enableLastfm"`
	EnableListenbrainz     bool     `json:"enableListenbrainz"`
	LastfmUsername         string   `json:"lastfmUsername"`
	ListenbrainzUsername   string   `json:"listenbrainzUsername"`
	FrameworkEnable        bool     `json:"frameworkEnable"`
	FrameworkModel         string   `json:"frameworkModel"`
	NemuEnable             bool     `json:"nemuEnable"`
	NemuGPUPassthrough     bool     `json:"nemuGpuPassthrough"`
	LUKSTPM2Enable         bool     `json:"luksTpm2Enable"`
	RecoveryEnable         bool     `json:"recoveryEnable"`
	JODSPrebootLockEnable  bool     `json:"jodsPrebootLockEnable"`
	SecureBootEnable       bool     `json:"secureBootEnable"`
	SecureBootPrompt       bool     `json:"secureBootPrompt"`
	EndpointManagedDevice  bool     `json:"endpointManagedDevice"`
	AutoReboot             bool     `json:"autoReboot"`
	RunUpdateChecks        bool     `json:"runUpdateChecks"`
	WriteConfig            bool     `json:"writeConfig"`
	RunRebuild             bool     `json:"runRebuild"`
}

var usernamePattern = regexp.MustCompile(`^[a-z_][a-z0-9_-]*$`)

func Load(path string) (User, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return User{}, fmt.Errorf("read user configuration: %w", err)
	}
	decoder := json.NewDecoder(strings.NewReader(string(contents)))
	decoder.DisallowUnknownFields()
	var user User
	if err := decoder.Decode(&user); err != nil {
		return User{}, fmt.Errorf("parse user configuration: %w", err)
	}
	if err := Validate(user); err != nil {
		return User{}, err
	}
	return user, nil
}

func Validate(user User) error {
	if !usernamePattern.MatchString(user.Username) {
		return fmt.Errorf("invalid username: %q", user.Username)
	}
	if user.WorkUserEnable {
		if user.WorkUsername != "" && !usernamePattern.MatchString(user.WorkUsername) {
			return fmt.Errorf("invalid work username: %q", user.WorkUsername)
		}
	}
	if user.System != "x86_64-linux" && user.System != "aarch64-linux" {
		return fmt.Errorf("unsupported system: %q", user.System)
	}
	if user.Profile == "" || user.Hostname == "" || user.Theme == "" || user.Shell == "" {
		return fmt.Errorf("profile, hostname, shell, and theme are required")
	}
	if len(user.Editors) == 0 || len(user.Browsers) == 0 {
		return fmt.Errorf("at least one editor and browser are required")
	}
	return nil
}
