package nixrender

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bakanura/gjallarOS/internal/installer/config"
)

func TestSyncUserIntentRetiresLegacyWebApplicationAssignments(t *testing.T) {
	path := filepath.Join(
		t.TempDir(),
		"state.nix",
	)

	oldState := `{pkgs, inputs, ...}:
rec {
    planeEnable = true;
    planeHost = "https://legacy-plane.example.test";
    drawioEnable = true;
    drawioSelfHosted = true;
    drawioHost = "https://legacy-drawio.example.test";

    derivedMarker = "preserve-me";

    themeDetails = {};
}
`

	if err := os.WriteFile(
		path,
		[]byte(oldState),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	user := config.User{
		WebApplications: []config.WebApplicationIntent{
			{
				ID: config.WebApplicationTeams,
			},
			{
				ID:       config.WebApplicationPlane,
				Endpoint: "https://plane.example.test",
			},
		},
	}

	if err := SyncUserIntent(path, user); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	state := string(data)

	for _, legacy := range []string{
		"planeEnable = ",
		"planeHost = ",
		"drawioEnable = ",
		"drawioSelfHosted = ",
		"drawioHost = ",
	} {
		if strings.Contains(state, legacy) {
			t.Fatalf(
				"legacy assignment %q survived sync:\n%s",
				legacy,
				state,
			)
		}
	}

	if !strings.Contains(
		state,
		`webApplications = [ { id = "teams"; endpoint = ""; } { id = "plane"; endpoint = "https://plane.example.test"; } ];`,
	) {
		t.Fatalf(
			"canonical webApplications was not inserted:\n%s",
			state,
		)
	}

	if !strings.Contains(
		state,
		`derivedMarker = "preserve-me";`,
	) {
		t.Fatalf(
			"unrelated generated state changed:\n%s",
			state,
		)
	}
}

func TestRetiredGeneratedKeysContainsLegacyWebApplicationAssignments(t *testing.T) {
	for _, key := range []string{
		"planeEnable",
		"planeHost",
		"drawioEnable",
		"drawioSelfHosted",
		"drawioHost",
	} {
		if !retiredGeneratedKeys[key] {
			t.Fatalf(
				"%s is not registered as retired generated state",
				key,
			)
		}
	}
}
