package nixrender

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bakanura/gjallarOS/internal/installer/config"
)

func TestStringEscapesNixInterpolation(t *testing.T) {
	got := String(`hello ${builtins.abort "no"} \\ "world"`)
	want := `"hello \${builtins.abort \"no\"} \\\\ \"world\""`
	if got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestRenderEscapesAllUserStrings(t *testing.T) {
	s := Settings{System: "x86_64-linux", Hostname: `${builtins.abort "bad"}`, Editors: []string{`a${b}`}}
	got := string(Render(s))
	if !strings.Contains(got, `hostname = "\${builtins.abort \"bad\"}";`) || !strings.Contains(got, `editors = [ "a\${b}" ];`) {
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
		DeviceSysVendor:      "HP",
		DeviceProductName:    "HP ZBook x2 G4",
		DeviceProductVersion: "A",
		DeviceBoardVendor:    "HP",
		DeviceBoardName:      "824C",
		DeviceBoardVersion:   "KBC Version 43.72",
	}))

	for _, want := range []string{
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

func TestRenderIncludesOrientationSensorSetting(t *testing.T) {
	rendered := string(Render(Settings{
		OrientationSensorEnable: true,
	}))

	if !strings.Contains(rendered, "orientationSensorEnable = true;") {
		t.Fatalf("orientation setting missing from render: %s", rendered)
	}
}

func TestRenderThemeDetailsUsesRepositoryThemeDirectory(t *testing.T) {
	got := string(Render(Settings{Theme: "noctalia"}))
	want := `themeDetails = import (./. + "/../themes/${theme}.nix") {inherit pkgs;};`
	if !strings.Contains(got, want) {
		t.Fatalf("theme path is not rooted above generated/:\n%s", got)
	}
}

func TestFromUserMapsRoutedIntent(t *testing.T) {
	user := config.User{
		Hostname:               "gjallarOS",
		Username:               "baka",
		PrintingEnable:         true,
		NetworkPrintingEnable:  true,
		ContainersEnable:       true,
		PlaneEnable:            true,
		PlaneHost:              "https://plane.example.test",
		EndpointManagedDevice:  true,
		JODSEndpoint:           "https://jods.example.test",
		JODSPolicySigningKey:   strings.Repeat("ab", 32),
		JODSRecoverySigningKey: strings.Repeat("cd", 32),
		JODSEnrollmentMode:     "manual",
		JODSDeviceClass:        "laptop",
		JODSDesktopProfile:     "headless",
		JODSFingerprintEnroll:  true,
		Theme:                  "noctalia",
		Editors:                []string{"vscodium"},
		Browsers:               []string{"librewolf"},
		PreferredEditor:        "vscodium",
		PreferredBrowser:       "librewolf",
		Shell:                  "zsh",
		AIAgentMode:            "workspace",
	}
	got := FromUser(user)
	if !got.PrintingEnable || !got.NetworkPrintingEnable || !got.ContainersEnable {
		t.Fatalf("routed booleans were not mapped: %#v", got)
	}
	if got.PlaneHost != user.PlaneHost || got.JODSEndpoint != user.JODSEndpoint || got.Theme != user.Theme {
		t.Fatalf("routed strings were not mapped: %#v", got)
	}
	if got.ODDCModel != "" || got.GraphicsVendor != "" || got.AIModel != "" {
		t.Fatalf("derived fields leaked into FromUser: %#v", got)
	}
}

func TestSyncUserIntentPreservesDerivedState(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.nix")
	before := `{pkgs, inputs, ...}:
rec {
    hostname = "old";
    printingEnable = false;
    planeEnable = false;
    planeHost = "";
    graphicsVendor = "amd";
    aiModel = "derived-model";
    theme = "old-theme";
    themeDetails = import (./. + "/../themes/${theme}.nix") {inherit pkgs;};
}
`
	if err := os.WriteFile(path, []byte(before), 0o640); err != nil {
		t.Fatal(err)
	}

	user := config.User{
		Hostname:           "gjallarOS",
		PrintingEnable:     true,
		PlaneEnable:        true,
		PlaneHost:          "https://plane.example.test",
		Theme:              "noctalia",
		AIAgentMode:        "workspace",
		JODSEnrollmentMode: "manual",
	}
	if err := SyncUserIntent(path, user); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	for _, want := range []string{
		`hostname = "gjallarOS";`,
		`printingEnable = true;`,
		`planeEnable = true;`,
		`planeHost = "https://plane.example.test";`,
		`graphicsVendor = "amd";`,
		`aiModel = "derived-model";`,
		`theme = "noctalia";`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q after sync:\n%s", want, got)
		}
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("mode changed to %o", info.Mode().Perm())
	}
}
