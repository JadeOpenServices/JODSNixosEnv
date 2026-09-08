package app

import (
	"os"
	"strings"
	"testing"
)

func TestFreshBareMetalPipelineIsWired(t *testing.T) {
	data, err := os.ReadFile("fresh_baremetal.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)

	ordered := []string{
		"targetdisk.Validate(",
		"freshdiskplan.Build(",
		"installconfirm.Confirm(",
		"freshgpt.Provision(",
		"rootprovision.Provision(",
		"mounttree.Prepare(",
		"stageFreshPasswordFiles(",
		"hardwareconfig.GenerateTarget(",
		"baremetalinstall.Install(",
	}

	last := -1
	for _, token := range ordered {
		pos := strings.Index(text, token)
		if pos < 0 {
			t.Fatalf("fresh pipeline missing %q", token)
		}
		if pos <= last {
			t.Fatalf(
				"fresh pipeline order invalid at %q",
				token,
			)
		}
		last = pos
	}
}

func TestFreshBareMetalSecretIsNotPlainPromptInput(t *testing.T) {
	data, err := os.ReadFile("fresh_baremetal.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)

	if !strings.Contains(
		text,
		`os.OpenFile("/dev/tty", os.O_RDWR, 0)`,
	) {
		t.Fatal("LUKS secret is not read from controlling terminal")
	}
	if !strings.Contains(
		text,
		"workpassword.ReadConfirmedPassword(",
	) {
		t.Fatal("confirmed no-echo secret reader is not used")
	}

	section := text[strings.Index(text, "func readFreshLUKSPassphrase"):]
	if strings.Contains(section, "ui.Value(") {
		t.Fatal("LUKS secret uses plain-text installer prompt")
	}
}

func TestFreshMountTreeOwnsCanonicalRecovery(t *testing.T) {
	data, err := os.ReadFile("fresh_baremetal.go")
	if err != nil {
		t.Fatal(err)
	}

	text := string(data)

	if strings.Contains(text, "mountPlan.Recovery = nil") {
		t.Fatal("fresh recovery partition is still stripped from canonical mounttree ownership")
	}

	if !strings.Contains(text, "Plan: plan") {
		t.Fatal("mounttree is not given the full canonical fresh disk plan")
	}

	if !strings.Contains(
		text,
		`const targetRoot = "/mnt/var/lib/gjallarOS/passwords"`,
	) {
		t.Fatal("fresh account password hashes are not staged into target root")
	}
}

func TestFreshInstallDoesNotRunHostLUKSMigration(t *testing.T) {
	data, err := os.ReadFile("app.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)

	if !strings.Contains(
		text,
		"Fresh-target TPM2 LUKS enrollment deferred until the installed GjallarOS lifecycle.",
	) {
		t.Fatal("fresh-target TPM/LUKS deferral marker missing")
	}

	legacy := `controlAttached(ctx, s.control, "installer", "luks", "--repo", root, "--hardware", hardwarePath)`
	if strings.Contains(text, legacy) {
		t.Fatal(
			"fresh installer still invokes host-side LUKS migration before target root exists",
		)
	}
}

func TestRecoveryFreshModeDoesNotTreatSourceRepoAsExistingInstall(t *testing.T) {
	appSource, err := os.ReadFile("app.go")
	if err != nil {
		t.Fatal(err)
	}

	body := string(appSource)

	if !strings.Contains(body, `if opt.recovery && !opt.acceptExisting`) {
		t.Fatal("recovery fresh operation does not explicitly override source-repository existing-install detection")
	}

	if !strings.Contains(body, `s.existing = false`) {
		t.Fatal("recovery fresh operation does not force fresh target state")
	}
}

func TestRecoveryAndAcceptExistingSelectReinstallMode(t *testing.T) {
	appSource, err := os.ReadFile("app.go")
	if err != nil {
		t.Fatal(err)
	}

	body := string(appSource)

	if strings.Contains(body, `--fresh-install and --accept-existing are mutually exclusive`) {
		t.Fatal("legacy fresh-install mutual exclusion remains")
	}

	if !strings.Contains(body, `if opt.recovery && !opt.acceptExisting`) {
		t.Fatal("recovery fresh/reinstall operation split is missing")
	}

	if !strings.Contains(body, `s.existing = existingInstall(root)`) {
		t.Fatal("recovery --accept-existing path no longer performs existing-install detection")
	}
}

func TestRecoveryEnvironmentSkipsLiveHostPreparation(t *testing.T) {
	appSource, err := os.ReadFile("app.go")
	if err != nil {
		t.Fatal(err)
	}

	body := string(appSource)

	if !strings.Contains(
		body,
		`Recovery environment validated; live-host rebuild skipped.`,
	) {
		t.Fatal("recovery environment is not explicitly separated from host preparation")
	}

	if !strings.Contains(body, `} else if code := prepareHost(`) {
		t.Fatal("normal installed-system host preparation was removed")
	}
}

func TestFreshInstallSkipsLocalGitProtection(t *testing.T) {
	appSource, err := os.ReadFile("app.go")
	if err != nil {
		t.Fatal(err)
	}

	body := string(appSource)

	if !strings.Contains(
		body,
		`Recovery installer source is embedded; local Git protection skipped.`,
	) {
		t.Fatal("fresh installer does not skip local Git protection")
	}

	if !strings.Contains(body, `localgit.Protect(`) {
		t.Fatal("normal installed-system local Git protection was removed")
	}
}

func TestRecoveryFlagSeparatesEnvironmentFromOperation(t *testing.T) {
	appSource, err := os.ReadFile("app.go")
	if err != nil {
		t.Fatal(err)
	}

	body := string(appSource)

	for _, want := range []string{
		`"recovery"`,
		`if opt.recovery && !opt.acceptExisting`,
		`Recovery environment validated; live-host rebuild skipped.`,
		`Recovery installer source is embedded; local Git protection skipped.`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing recovery execution-environment contract %q", want)
		}
	}

	for _, forbidden := range []string{
		`"fresh-install"`,
		`opt.freshInstall`,
	} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("legacy fresh-install flag contract remains: %q", forbidden)
		}
	}
}
