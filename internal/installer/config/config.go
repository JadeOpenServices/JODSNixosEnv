// Package config defines the typed contract between installer input,
// discovery, and the generated settings.nix file.
package config

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type User struct {
	Profile                   string   `json:"profile"`
	Hostname                  string   `json:"hostname"`
	Username                  string   `json:"username"`
	Timezone                  string   `json:"timezone"`
	Locale                    string   `json:"locale"`
	KeyboardLayout            string   `json:"keyboardLayout"`
	KeyboardVariant           string   `json:"keyboardVariant"`
	WeatherCity               string   `json:"weatherCity"`
	WeatherCountry            string   `json:"weatherCountry"`
	TouchpadWorkspaceSwipe    bool     `json:"touchpadWorkspaceSwipe"`
	ClamshellEnable           bool     `json:"clamshellEnable"`
	USBGuardEnable            bool     `json:"usbguardEnable"`
	Name                      string   `json:"name"`
	Email                     string   `json:"email"`
	GitHubUsername            string   `json:"githubUsername"`
	DotfilesDir               string   `json:"dotfilesDir"`
	Shell                     string   `json:"shell"`
	Editors                   []string `json:"editors"`
	Browsers                  []string `json:"browsers"`
	PreferredEditor           string   `json:"preferredEditor"`
	PreferredBrowser          string   `json:"preferredBrowser"`
	PlaneEnable               bool     `json:"planeEnable"`
	PlaneHost                 string   `json:"planeHost"`
	DrawioEnable              bool     `json:"drawioEnable"`
	DrawioSelfHosted          bool     `json:"drawioSelfHosted"`
	DrawioHost                string   `json:"drawioHost"`
	Theme                     string   `json:"theme"`
	BackgroundNormal          string   `json:"backgroundNormal"`
	BackgroundWork            string   `json:"backgroundWork"`
	BackgroundGaming          string   `json:"backgroundGaming"`
	WorkUserEnable            bool     `json:"workUserEnable"`
	WorkUsername              string   `json:"workUsername"`
	WorkUserPasswordFile      string   `json:"workUserPasswordFile"`
	DockerEnable              bool     `json:"dockerEnable"`
	DebugFunctions            bool     `json:"debugFunctions"`
	AIEnable                  bool     `json:"aiEnable"`
	OverrideAISelection       bool     `json:"overrideAiSelection"`
	OverrideModelWith         string   `json:"overrideModelWith"`
	AIAgentMode               string   `json:"aiAgentMode"`
	EnableScrobbling          bool     `json:"enableScrobbling"`
	EnableLastfm              bool     `json:"enableLastfm"`
	EnableListenbrainz        bool     `json:"enableListenbrainz"`
	LastfmUsername            string   `json:"lastfmUsername"`
	ListenbrainzUsername      string   `json:"listenbrainzUsername"`
	NemuEnable                bool     `json:"nemuEnable"`
	NemuGPUPassthrough        bool     `json:"nemuGpuPassthrough"`
	LUKSTPM2Enable            bool     `json:"luksTpm2Enable"`
	RecoveryEnable            bool     `json:"recoveryEnable"`
	RecoveryPartitionEnable   bool     `json:"recoveryPartitionEnable"`
	JODSPrebootLockEnable     bool     `json:"jodsPrebootLockEnable"`
	SecureBootEnable          bool     `json:"secureBootEnable"`
	SecureBootPrompt          bool     `json:"secureBootPrompt"`
	EndpointManagedDevice     bool     `json:"endpointManagedDevice"`
	JODSEndpoint              string   `json:"jodsEndpoint"`
	JODSPolicySigningKey      string   `json:"jodsPolicySigningPublicKey"`
	JODSRecoverySigningKey    string   `json:"jodsRecoveryCommandSigningPublicKey"`
	JODSEnrollmentMode        string   `json:"jodsEnrollmentMode"`
	JODSAllowInsecureTLS      bool     `json:"jodsAllowInsecureTls"`
	JODSDeviceClass           string   `json:"jodsDeviceClass"`
	JODSDesktopProfile        string   `json:"jodsDesktopProfile"`
	JODSFingerprintEnroll     bool     `json:"jodsFingerprintEnrollmentAllowed"`
	AllowUnvalidatedODDCModel bool     `json:"allowUnvalidatedODDCModel"`
	UnattendedInstall         bool     `json:"unattendedInstall"`
	AutoReboot                bool     `json:"autoReboot"`
	RunUpdateChecks           bool     `json:"runUpdateChecks"`
	WriteConfig               bool     `json:"writeConfig"`
	RunRebuild                bool     `json:"runRebuild"`
	ForceRedeploy             bool     `json:"forceRedeploy"`
}

// WriteAtomic persists the confirmed machine-local installer input.
func WriteAtomic(path string, user User) error {
	data, err := json.MarshalIndent(user, "", "  ")
	if err != nil {
		return fmt.Errorf("encode user configuration: %w", err)
	}
	data = append(data, '\n')

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".user.config.json-*")
	if err != nil {
		return fmt.Errorf("create temporary user configuration: %w", err)
	}

	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return fmt.Errorf("set user configuration permissions: %w", err)
	}

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write user configuration: %w", err)
	}

	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync user configuration: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close user configuration: %w", err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace user configuration: %w", err)
	}

	return nil
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
	if err := NormalizeProjectTools(&user); err != nil {
		return User{}, err
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
	if user.Profile == "" || user.Hostname == "" || user.Theme == "" || user.Shell == "" {
		return fmt.Errorf("profile, hostname, shell, and theme are required")
	}
	if len(user.Editors) == 0 || len(user.Browsers) == 0 {
		return fmt.Errorf("at least one editor and browser are required")
	}
	if user.AIEnable {
		if user.OverrideAISelection && strings.TrimSpace(user.OverrideModelWith) == "" {
			return fmt.Errorf("overrideModelWith is required when overrideAiSelection is true")
		}
		if !oneOf(user.AIAgentMode, "workspace", "owner-conservative", "owner-full-local") {
			return fmt.Errorf("invalid aiAgentMode: %q", user.AIAgentMode)
		}
	}
	if err := ValidateJODS(user); err != nil {
		return err
	}
	if err := ValidateProjectTools(user); err != nil {
		return err
	}

	return nil
}

func NormalizeExternalServiceEndpoint(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("external service endpoint is empty")
	}

	if !strings.Contains(value, "://") {
		value = "http://" + value
	}

	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || parsed.User != nil {
		return "", fmt.Errorf("malformed external service endpoint %q", raw)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("unsupported external service endpoint scheme %q", parsed.Scheme)
	}

	// Project-tool endpoints are canonical base URLs. Keep an explicit
	// scheme and port unchanged, while removing redundant trailing slashes.
	// Remove redundant trailing path slashes for ordinary base URLs.
	// Keep the root slash when a query/fragment is present so endpoints such as
	// http://host:8080/?offline=1&https=0 remain intact.
	if parsed.RawQuery == "" && parsed.Fragment == "" {
		parsed.Path = strings.TrimRight(parsed.Path, "/")
	} else if parsed.Path != "/" {
		parsed.Path = strings.TrimRight(parsed.Path, "/")
	}

	return strings.TrimRight(parsed.String(), "/"), nil
}

func NormalizeProjectTools(user *User) error {
	if user.PlaneEnable {
		normalized, err := NormalizeExternalServiceEndpoint(user.PlaneHost)
		if err != nil {
			return fmt.Errorf("planeHost: %w", err)
		}
		user.PlaneHost = normalized
	}

	if user.DrawioEnable && user.DrawioSelfHosted {
		normalized, err := NormalizeExternalServiceEndpoint(user.DrawioHost)
		if err != nil {
			return fmt.Errorf("drawioHost: %w", err)
		}
		user.DrawioHost = normalized
	} else if user.DrawioEnable {
		// Public diagrams.net mode does not require or consume a custom host.
		user.DrawioHost = ""
	}

	return nil
}

func ValidateProjectTools(user User) error {
	return NormalizeProjectTools(&user)
}

var jodsPublicKeyPattern = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)
var jodsDesktopProfilePattern = regexp.MustCompile(`^[A-Za-z0-9._+:-]{1,64}$`)

func ValidateJODS(user User) error {
	if !user.EndpointManagedDevice {
		if user.JODSFingerprintEnroll {
			return fmt.Errorf("JODS fingerprint enrollment policy requires endpoint management enrollment")
		}
		if user.JODSPrebootLockEnable {
			return fmt.Errorf("JODS preboot locking requires explicit JODS endpoint management enrollment")
		}
		return nil
	}
	parsed, err := url.Parse(user.JODSEndpoint)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return fmt.Errorf("JODS endpoint must be an HTTPS URL without embedded credentials")
	}
	if !jodsPublicKeyPattern.MatchString(user.JODSPolicySigningKey) {
		return fmt.Errorf("JODS policy-signing public key must be exactly 64 hexadecimal characters")
	}
	if !jodsPublicKeyPattern.MatchString(user.JODSRecoverySigningKey) {
		return fmt.Errorf("JODS recovery-command signing public key must be exactly 64 hexadecimal characters")
	}
	if strings.EqualFold(user.JODSPolicySigningKey, user.JODSRecoverySigningKey) {
		return fmt.Errorf("JODS policy and recovery-command signing keys must differ")
	}
	if !oneOf(user.JODSEnrollmentMode, "auto", "manual", "jade-registry-only") {
		return fmt.Errorf("invalid JODS enrollment mode: %q", user.JODSEnrollmentMode)
	}
	if !oneOf(user.JODSDeviceClass, "pc", "vm", "laptop", "kiosk", "workstation") {
		return fmt.Errorf("invalid JODS device class: %q", user.JODSDeviceClass)
	}
	if !jodsDesktopProfilePattern.MatchString(strings.TrimSpace(user.JODSDesktopProfile)) {
		return fmt.Errorf("invalid JODS desktop profile: %q", user.JODSDesktopProfile)
	}
	host := parsed.Hostname()
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return fmt.Errorf("JODS endpoint must not use a loopback address")
	}
	return nil
}

func oneOf(value string, allowed ...string) bool {
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}
