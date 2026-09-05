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
		"jodsPrebootLockEnable = false;",
		"secureBootEnable = false;",
		"endpointManagedDevice = false;",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q from settings:\n%s", want, got)
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
		EndpointManagedDevice:      true,
		JODSEndpoint:               "https://jods.example.test:1666",
		JODSPolicySigningPublicKey: strings.Repeat("ab", 32),
		JODSEnrollmentMode:         "jade-registry-only",
		JODSAllowInsecureTLS:       false,
		JODSDeviceClass:            "laptop",
		JODSDesktopProfile:         "headless",
	}))
	for _, want := range []string{
		"endpointManagedDevice = true;",
		`jodsEndpoint = "https://jods.example.test:1666";`,
		`jodsEnrollmentMode = "jade-registry-only";`,
		"jodsAllowInsecureTls = false;",
		`jodsDeviceClass = "laptop";`,
		`jodsDesktopProfile = "headless";`,
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
