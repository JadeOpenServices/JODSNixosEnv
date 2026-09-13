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
		"materializeDeviceProfileCapsule",
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

	for _, want := range []string{
		`if opt.recovery {`,
		`if opt.acceptExisting {`,
		`s.existing = false`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("recovery fresh operation contract missing %q", want)
		}
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

	for _, want := range []string{
		`if opt.recovery {`,
		`if opt.acceptExisting {`,
		`installedRoot = "/mnt"`,
		`detectRecoveryInstalledRoot(ctx, installedRoot)`,
		`recoverytarget.PrepareBoot(`,
		`installedRoot,`,
		`baremetalinstall.Install(`,
		`Repo:     root,`,
		`Hostname: s.user.Hostname,`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("recovery fresh/reinstall operation contract missing %q", want)
		}
	}

	if !strings.Contains(body, `detectExistingInstalledSystem(ctx, root)`) {
		t.Fatal("normal installed-system detection was removed")
	}

	recoveryInstall := strings.Index(
		body,
		`if opt.recovery && opt.acceptExisting {`,
	)
	if recoveryInstall < 0 {
		t.Fatal("recovery reinstall deployment branch is missing")
	}

	recoveryPrepare := strings.Index(
		body[recoveryInstall:],
		`recoverytarget.PrepareBoot(`,
	)
	recoveryDeploy := strings.Index(
		body[recoveryInstall:],
		`baremetalinstall.Install(`,
	)
	normalDeploy := strings.Index(
		body[recoveryInstall:],
		`deploy.Target(`,
	)

	if recoveryPrepare < 0 || recoveryDeploy < 0 || normalDeploy < 0 {
		t.Fatal("recovery and normal deployment boundaries are incomplete")
	}

	if !(recoveryPrepare < recoveryDeploy && recoveryDeploy < normalDeploy) {
		t.Fatal(
			"recovery reinstall must prepare /mnt and use baremetalinstall before the normal deploy path",
		)
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
		`if opt.recovery {`,
		`if opt.acceptExisting {`,
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

func TestRecoveryHardwareRebindRematerializesAfterHardwareReconciliation(t *testing.T) {
	appSource, err := os.ReadFile("app.go")
	if err != nil {
		t.Fatal(err)
	}

	body := string(appSource)

	reconcile := strings.Index(
		body,
		"reconcileHardwareConfiguration(",
	)
	if reconcile < 0 {
		t.Fatal("hardware reconciliation call is missing")
	}

	afterReconcile := body[reconcile:]

	rebindGateRel := strings.Index(
		afterReconcile,
		"if needsDeviceRebind {",
	)
	if rebindGateRel < 0 {
		t.Fatal("post-reconciliation device rebind gate is missing")
	}
	rebindGate := reconcile + rebindGateRel

	rebindMaterializeRel := strings.Index(
		body[rebindGate:],
		"materializeDeviceProfileCapsule(",
	)
	if rebindMaterializeRel < 0 {
		t.Fatal("device rebind does not rematerialize the recovery capsule")
	}
	rebindMaterialize := rebindGate + rebindMaterializeRel

	rebindReasonRel := strings.Index(
		body[rebindMaterialize:],
		`"hardware-rebind"`,
	)
	if rebindReasonRel < 0 {
		t.Fatal("hardware-rebind generation reason is missing")
	}
	rebindReason := rebindMaterialize + rebindReasonRel

	if !(reconcile < rebindGate &&
		rebindGate < rebindMaterialize &&
		rebindMaterialize < rebindReason) {
		t.Fatal(
			"recovery capsule rebind must occur after hardware reconciliation and behind the rebind gate",
		)
	}
}

func TestRecoverySameMachineUsesVerifiedCachedODDCSource(t *testing.T) {
	data, err := os.ReadFile("app.go")
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)

	verify := strings.Index(body, "deviceprofilecache.Verify(capsulePath)")
	cached := strings.Index(body, "filepath.Join(capsulePath, \"oddc\")")
	persist := strings.Index(body, "persistDeviceIdentity(&s.user, hardware, resolvedDevice)")

	if verify < 0 || cached < 0 || persist < 0 {
		t.Fatal("cached recovery ODDC source flow is incomplete")
	}
	if !(verify < cached && cached < persist) {
		t.Fatal("same-machine recovery must verify and resolve cached ODDC before persisting device identity")
	}

	if !strings.Contains(body[verify:persist], "if !needsDeviceRebind {") {
		t.Fatal("cached ODDC source is not gated to the same-machine recovery path")
	}
}

func TestRecoveryHardwareRebindCommitsCapsuleAfterInstall(t *testing.T) {
	data, err := os.ReadFile("app.go")
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)

	install := strings.Index(body, "baremetalinstall.Install(")
	reason := strings.Index(body, "\"hardware-rebind\"")

	if install < 0 || reason < 0 {
		t.Fatal("recovery rebind transaction is incomplete")
	}
	if install >= reason {
		t.Fatal("hardware-rebind capsule must be committed only after successful target install")
	}
}

func TestRecoveryMaintenanceBootUsesInstalledRoot(t *testing.T) {
	body, err := os.ReadFile("app.go")
	if err != nil {
		t.Fatal(err)
	}

	text := string(body)
	call := strings.Index(text, "\"gjallar-recovery-maintenance-next\",")
	if call < 0 {
		t.Fatal("maintenance boot helper invocation is missing")
	}

	window := text[call:]
	if !strings.Contains(window, "installedRoot") {
		t.Fatal("maintenance boot helper is not passed installedRoot")
	}
}

func TestRecoveryHardwareRebindCannotSkipInstall(t *testing.T) {
	data, err := os.ReadFile("app.go")
	if err != nil {
		t.Fatal(err)
	}
	body := string(data)

	want := "if (profileDrift || needsDeviceRebind) && !runRebuild {"
	if !strings.Contains(body, want) {
		t.Fatal("device profile reconciliation can still skip target installation")
	}
}

func TestDeviceProfileDriftCannotSkipInstall(t *testing.T) {
	data, err := os.ReadFile("app.go")
	if err != nil {
		t.Fatal(err)
	}

	body := string(data)

	want := "if (profileDrift || needsDeviceRebind) && !runRebuild {"
	if !strings.Contains(body, want) {
		t.Fatal("device profile drift can still skip activation")
	}

	if !strings.Contains(
		body,
		"device profile reconciliation requires installing the regenerated system",
	) {
		t.Fatal("generic device profile reconciliation refusal is missing")
	}
}

func TestForceRedeployPreservesExistingInstallState(t *testing.T) {
	appSource, err := os.ReadFile("app.go")
	if err != nil {
		t.Fatal(err)
	}

	body := string(appSource)

	for _, want := range []string{
		`f.BoolVar(&opt.forceRedeploy, "force-redeploy"`,
		`forceRedeploy := opt.forceRedeploy || s.user.ForceRedeploy`,
		`opt.forceRedeploy = forceRedeploy`,
		`if opt.forceRedeploy {`,
		`if s.existing && !opt.forceRedeploy {`,
		`Force redeployment requested; running prerequisite bootstrap despite existing-install detection.`,
		`--force-redeploy cannot be combined with --no-rebuild`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("force-redeploy lifecycle boundary missing %q", want)
		}
	}

	start := strings.Index(body, "func prepareHost(")
	if start < 0 {
		t.Fatal("prepareHost is missing")
	}

	prepareHost := body[start:]
	if next := strings.Index(prepareHost[1:], "\nfunc "); next >= 0 {
		prepareHost = prepareHost[:next+1]
	}

	if strings.Contains(prepareHost, "s.existing = false") {
		t.Fatal("force redeployment must not erase existing-install state")
	}

	if strings.Contains(body, "forceBootstrap") ||
		strings.Contains(body, "force-bootstrap") {
		t.Fatal("obsolete force-bootstrap lifecycle remains")
	}
}

func TestForceRedeployIsNonDestructive(t *testing.T) {
	appSource, err := os.ReadFile("app.go")
	if err != nil {
		t.Fatal(err)
	}

	body := string(appSource)

	want := "Force clean redeployment selected; preserving disk layout, credentials, encryption keys, and user data."
	if !strings.Contains(body, want) {
		t.Fatal("force redeployment preservation boundary is missing")
	}
}
