package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bakanura/gjallarOS/internal/installer/config"
	"github.com/bakanura/gjallarOS/internal/installercheck"
	"github.com/bakanura/gjallarOS/internal/oddccli"
)

func TestRebuildDoesNotRequireExplicitRepoAndHost(t *testing.T) {
	t.Setenv("GJALLAROS_REPO", "")
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	t.Chdir(t.TempDir())
	oldSystem := installercheck.SystemRepositoryFile
	installercheck.SystemRepositoryFile = filepath.Join(t.TempDir(), "repository")
	t.Cleanup(func() { installercheck.SystemRepositoryFile = oldSystem })

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	status := runRebuild(nil, &stdout, &stderr)

	if status != 2 {
		t.Fatalf(
			"runRebuild status = %d, want 2 for unresolved repository",
			status,
		)
	}

	if strings.Contains(
		stderr.String(),
		"rebuild requires --repo and --host",
	) {
		t.Fatalf(
			"rebuild still requires explicit repo/host:\n%s",
			stderr.String(),
		)
	}

	if !strings.Contains(
		stderr.String(),
		"resolve GjallarOS repository",
	) {
		t.Fatalf(
			"rebuild did not reach repository resolution:\n%s",
			stderr.String(),
		)
	}
}

func TestRebuildPreflightSuccessIsCompact(t *testing.T) {
	findings := []installercheck.Finding{
		{
			Level:   "PASS",
			Message: "default preset is valid",
		},
		{
			Level:   installercheck.Warn,
			Message: "known optional warning",
		},
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	status := reportRebuildPreflightResult(
		findings,
		false,
		&stdout,
		&stderr,
	)

	if status != 0 {
		t.Fatalf("status = %d, want 0", status)
	}

	got := stdout.String()

	if !strings.Contains(got, "[GjallarOS] Preflight ✓") {
		t.Fatalf("compact success line missing:\n%s", got)
	}

	if !strings.Contains(got, "WARN: known optional warning") {
		t.Fatalf("warning was hidden:\n%s", got)
	}

	if strings.Contains(got, "PASS: default preset is valid") {
		t.Fatalf("successful finding leaked into rebuild output:\n%s", got)
	}

	if strings.Contains(got, "Fast preflight") {
		t.Fatalf("verbose preflight heading leaked into rebuild:\n%s", got)
	}

	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr:\n%s", stderr.String())
	}
}

func TestRebuildPreflightFailureKeepsDiagnostics(t *testing.T) {
	findings := []installercheck.Finding{
		{
			Level:   "PASS",
			Message: "good check",
		},
		{
			Level:   installercheck.Warn,
			Message: "warning detail",
		},
		{
			Level:   "FAIL",
			Message: "failure detail",
		},
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	status := reportRebuildPreflightResult(
		findings,
		true,
		&stdout,
		&stderr,
	)

	if status != 1 {
		t.Fatalf("status = %d, want 1", status)
	}

	got := stdout.String()

	if strings.Contains(got, "PASS: good check") {
		t.Fatalf("PASS noise leaked into failed rebuild:\n%s", got)
	}

	if !strings.Contains(got, "WARN: warning detail") {
		t.Fatalf("warning detail missing:\n%s", got)
	}

	if !strings.Contains(got, "FAIL: failure detail") {
		t.Fatalf("failure detail missing:\n%s", got)
	}

	if !strings.Contains(
		stderr.String(),
		"FAIL: GjallarOS preflight failed",
	) {
		t.Fatalf(
			"failure summary missing:\n%s",
			stderr.String(),
		)
	}
}

func TestNormalizeRebuildUserIntentMigratesBlankDotfilesDir(t *testing.T) {
	user := config.User{}

	changed, err := normalizeRebuildUserIntent(&user, "/home/test/gjallarOS")
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("blank dotfilesDir was not migrated")
	}
	if user.DotfilesDir != "/home/test/gjallarOS" {
		t.Fatalf("dotfilesDir=%q", user.DotfilesDir)
	}
}

func TestNormalizeRebuildUserIntentPreservesConfiguredDotfilesDir(t *testing.T) {
	user := config.User{DotfilesDir: "/srv/gjallarOS"}

	changed, err := normalizeRebuildUserIntent(&user, "/home/test/gjallarOS")
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("configured dotfilesDir was unexpectedly changed")
	}
	if user.DotfilesDir != "/srv/gjallarOS" {
		t.Fatalf("dotfilesDir=%q", user.DotfilesDir)
	}
}

func TestRebuildCommandArgsArePureWithoutPalette(t *testing.T) {
	args := rebuildCommandArgs(
		"/repo",
		"host",
		t.TempDir(),
		false,
		[]string{"--show-trace-extra"},
	)

	if got := countString(args, "--impure"); got != 0 {
		t.Fatalf("--impure count = %d, want 0: %v", got, args)
	}
	if got := countString(args, "--show-trace-extra"); got != 1 {
		t.Fatalf("forwarded rebuild arg count = %d, want 1: %v", got, args)
	}
}

func TestRebuildCommandArgsAddOneImpureForNoctaliaPalette(t *testing.T) {
	home := t.TempDir()
	palette := filepath.Join(
		home,
		".local",
		"state",
		"noctalia",
		"stylix-override.json",
	)
	if err := os.MkdirAll(filepath.Dir(palette), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(palette, []byte("{}\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	args := rebuildCommandArgs("/repo", "host", home, true, nil)

	if got := countString(args, "--impure"); got != 1 {
		t.Fatalf("--impure count = %d, want 1: %v", got, args)
	}
	if got := countString(args, "--show-trace"); got != 1 {
		t.Fatalf("--show-trace count = %d, want 1: %v", got, args)
	}
	if len(args) < 2 || args[0] != "env" || !strings.HasPrefix(args[1], "GJALLAR_NOCTALIA_PALETTE=") {
		t.Fatalf("palette environment prefix missing: %v", args)
	}
}

func TestRebuildLocalDerivationsSummarizesUniqueBuilds(t *testing.T) {
	log, err := os.CreateTemp(t.TempDir(), "rebuild-*.log")
	if err != nil {
		t.Fatal(err)
	}
	defer log.Close()

	input := strings.Join([]string{
		"these 3 derivations will be built:",
		"building '/nix/store/aaaaaaaa-noctalia-5.0.0.drv'...",
		"building '/nix/store/bbbbbbbb-home-manager-generation.drv'...",
		"building '/nix/store/aaaaaaaa-noctalia-5.0.0.drv'...",
		"copying path '/nix/store/cccccccc-not-a-local-build' from cache...",
	}, "\n")
	if _, err := log.WriteString(input); err != nil {
		t.Fatal(err)
	}

	got := rebuildLocalDerivations(log)
	want := []string{"noctalia-5.0.0", "home-manager-generation"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("builds = %v, want %v", got, want)
	}
}

func TestSystemGenerationCleanupArgsUseCountRetention(t *testing.T) {
	got := systemGenerationCleanupArgs(5)
	want := []string{
		"nix-env",
		"--profile",
		"/nix/var/nix/profiles/system",
		"--delete-generations",
		"+5",
	}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("cleanup args = %v, want %v", got, want)
	}
}

func TestParseCleanupKeepRejectsZeroAndGarbage(t *testing.T) {
	for _, value := range []string{"", "0", "-1", "five"} {
		if _, err := parseCleanupKeep(value); err == nil {
			t.Fatalf("parseCleanupKeep(%q) unexpectedly succeeded", value)
		}
	}

	got, err := parseCleanupKeep("7")
	if err != nil {
		t.Fatal(err)
	}
	if got != 7 {
		t.Fatalf("keep = %d, want 7", got)
	}
}

func countString(values []string, want string) int {
	count := 0
	for _, value := range values {
		if value == want {
			count++
		}
	}
	return count
}

func TestRebuildHardwareUpdateFailureStopsRebuild(t *testing.T) {
	repo, _ := filepath.EvalSymlinks(t.TempDir())
	for _, marker := range []string{"flake.nix", "scripts/installation/install.sh"} {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(repo, marker)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(repo, marker), nil, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	log := filepath.Join(t.TempDir(), "log")
	fake := filepath.Join(t.TempDir(), "oddc")
	body := "#!/bin/sh\necho \"$@\" > " + log + "\nexit 3\n"
	if err := os.WriteFile(fake, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	old := oddccli.Path
	oddccli.Path = fake
	t.Cleanup(func() { oddccli.Path = old })

	var stdout, stderr bytes.Buffer
	status := runRebuild(
		[]string{"--repo", repo, "--hardware-update", "--stage", "staging"},
		&stdout,
		&stderr,
	)

	if status != 3 {
		t.Fatalf("status = %d, want 3:\n%s", status, stderr.String())
	}
	ran, _ := os.ReadFile(log)
	if want := "update --flake " + repo + " --stage staging\n"; string(ran) != want {
		t.Fatalf("oddc ran %q, want %q", ran, want)
	}
	if !strings.Contains(stderr.String(), "nothing rebuilt") {
		t.Fatalf("failure not reported:\n%s", stderr.String())
	}
}

func TestRebuildStageRequiresHardwareUpdate(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if status := runRebuild([]string{"--repo", t.TempDir(), "--stage", "main"}, &stdout, &stderr); status != 2 {
		t.Fatalf("status = %d, want 2", status)
	}
	if status := runRebuild([]string{"--hardware-update", "--stage", "dev"}, &stdout, &stderr); status != 2 {
		t.Fatalf("bad stage status = %d, want 2", status)
	}
}
