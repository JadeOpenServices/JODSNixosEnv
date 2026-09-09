package nixrender

import (
	"strings"
	"testing"
)

func TestStringEscapesNixInterpolation(t *testing.T) {
	got := String(`hello ${builtins.abort "no"} \\ "world"`)
	want := `"hello \${builtins.abort \"no\"} \\\\ \"world\""`
	if got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestRenderEscapesAllUserStrings(t *testing.T) {
	s := Settings{System: "x86_64-linux", Profile: `${builtins.abort "bad"}`, Editors: []string{`a${b}`}}
	got := string(Render(s))
	if !strings.Contains(got, `profile = "\${builtins.abort \"bad\"}";`) || !strings.Contains(got, `editors = [ "a\${b}" ];`) {
		t.Fatalf("unsafe or missing escaped output:\n%s", got)
	}
}

func TestStrings(t *testing.T) {
	if got := Strings([]string{"a", "b"}); got != `[ "a" "b" ]` {
		t.Fatalf("Strings() = %q", got)
	}
}

func TestRenderRecoveryPolicyDefaultsDisabled(t *testing.T) {
	got := string(Render(Settings{}))
	for _, want := range []string{
		"recoveryEnable = false;",
		"recoveryPartitionEnable = false;",
		"jodsPrebootLockEnable = false;",
		"secureBootEnable = false;",
		"endpointManagedDevice = false;",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q from settings:\n%s", want, got)
		}
	}
}

func TestRenderAIResolvedState(t *testing.T) {
	got := string(Render(Settings{AIEnable: true, AIModel: "qwen2.5-coder:7b", AIAccelerationProfile: "auto", AIAgentMode: "workspace", AIContextTokens: 8192}))
	for _, want := range []string{"aiEnable = true;", `aiModel = "qwen2.5-coder:7b";`, `aiAccelerationProfile = "auto";`, `aiAgentMode = "workspace";`, "aiContextTokens = 8192;"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q", want)
		}
	}
}

func TestRenderWeatherLocation(t *testing.T) {
	got := string(Render(Settings{WeatherCity: "Berlin", WeatherCountry: "Germany"}))
	for _, want := range []string{`weatherCity = "Berlin";`, `weatherCountry = "Germany";`} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q from settings:\n%s", want, got)
		}
	}
}

func TestRenderJODSSettings(t *testing.T) {
	got := string(Render(Settings{
		EndpointManagedDevice:               true,
		JODSEndpoint:                        "https://jods.example.test:1666",
		JODSPolicySigningPublicKey:          strings.Repeat("ab", 32),
		JODSRecoveryCommandSigningPublicKey: strings.Repeat("cd", 32),
		JODSEnrollmentMode:                  "jade-registry-only",
		JODSAllowInsecureTLS:                false,
		JODSDeviceClass:                     "laptop",
		JODSDesktopProfile:                  "headless",
	}))
	for _, want := range []string{
		"endpointManagedDevice = true;",
		`jodsEndpoint = "https://jods.example.test:1666";`,
		`jodsRecoveryCommandSigningPublicKey = "cdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcd";`,
		`jodsEnrollmentMode = "jade-registry-only";`,
		"jodsAllowInsecureTls = false;",
		`jodsDeviceClass = "laptop";`,
		`jodsDesktopProfile = "headless";`,
		"jodsFingerprintEnrollmentAllowed = false;",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q from settings:\n%s", want, got)
		}
	}
}

func TestRenderJODSDoesNotContainSecretFields(t *testing.T) {
	got := string(Render(Settings{}))
	for _, forbidden := range []string{"devicePrivateKey", "apiToken", "luksRecoveryKey"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("generated settings contain secret field %q", forbidden)
		}
	}
}

func TestRenderDeviceIdentity(t *testing.T) {
	got := string(Render(Settings{
		DeviceProfile:        "laptop/common",
		DeviceSysVendor:      "HP",
		DeviceProductName:    "HP ZBook x2 G4",
		DeviceProductVersion: "A",
		DeviceBoardVendor:    "HP",
		DeviceBoardName:      "824C",
		DeviceBoardVersion:   "KBC Version 43.72",
	}))

	for _, want := range []string{
		`deviceProfile = "laptop/common";`,
		`deviceSysVendor = "HP";`,
		`deviceProductName = "HP ZBook x2 G4";`,
		`deviceProductVersion = "A";`,
		`deviceBoardVendor = "HP";`,
		`deviceBoardName = "824C";`,
		`deviceBoardVersion = "KBC Version 43.72";`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in:\n%s", want, got)
		}
	}
}
