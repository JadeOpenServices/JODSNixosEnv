package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/bakanura/gjallarOS/internal/ai/profile"
	"github.com/bakanura/gjallarOS/internal/hardware/graphics"
	"github.com/bakanura/gjallarOS/internal/input/xkb"
	"github.com/bakanura/gjallarOS/internal/installer/background"
	"github.com/bakanura/gjallarOS/internal/installer/baremetalinstall"
	"github.com/bakanura/gjallarOS/internal/installer/bootstrap"
	"github.com/bakanura/gjallarOS/internal/installer/config"
	"github.com/bakanura/gjallarOS/internal/installer/deploy"
	"github.com/bakanura/gjallarOS/internal/installer/deviceprofilecache"
	"github.com/bakanura/gjallarOS/internal/installer/discovery"
	"github.com/bakanura/gjallarOS/internal/installer/diskcrypto"
	"github.com/bakanura/gjallarOS/internal/installer/firmware"
	"github.com/bakanura/gjallarOS/internal/installer/geolocation"
	"github.com/bakanura/gjallarOS/internal/installer/hardwareconfig"
	"github.com/bakanura/gjallarOS/internal/installer/localgit"
	"github.com/bakanura/gjallarOS/internal/installer/nixrender"
	"github.com/bakanura/gjallarOS/internal/installer/oddc"
	"github.com/bakanura/gjallarOS/internal/installer/prompt"
	"github.com/bakanura/gjallarOS/internal/installer/recoveryprovision"
	"github.com/bakanura/gjallarOS/internal/installer/recoveryresize"
	"github.com/bakanura/gjallarOS/internal/installer/recoverytarget"
	"github.com/bakanura/gjallarOS/internal/installer/release"
	installerresume "github.com/bakanura/gjallarOS/internal/installer/resume"
	"github.com/bakanura/gjallarOS/internal/installer/secureboot"
	"github.com/bakanura/gjallarOS/internal/installer/sourcerevision"
	"github.com/bakanura/gjallarOS/internal/installercheck"
)

type options struct {
	recovery                                                 bool
	repo                                                     string
	skipHardware, refreshHardware, noRebuild, acceptExisting bool
	forceRedeploy                                            bool
	targetDisk                                               string
	recoveryDisk, recoveryPartition, recoverySigningKey      string
	recoverySigningPublicKey                                 string
}
type state struct {
	user                     config.User
	render                   nixrender.Settings
	preset, existing         bool
	control                  string
	touchscreen              bool
	penTablet                bool
	orientationSensor        bool
	recoveryDisk             string
	recoveryPartition        string
	recoverySigningKey       string
	recoverySigningPublicKey string
	secureBootFirmware       oddc.EffectiveSecureBootFirmwarePolicy
}

const (
	installerSecureBootResumeMarker = "/var/lib/gjallarOS/installer-resume-after-secure-boot"
	installerReleaseRebootScheduled = 75
)

var detectNetworkLocation = geolocation.Detect
var runJODSCommand = attached

func Run(ctx context.Context, args []string, in io.Reader, out, errOut io.Writer) int {
	if len(args) == 2 && args[0] == "--resume-transaction" {
		tx, err := installerresume.Load(args[1])
		if err != nil {
			return fail(errOut, err)
		}

		expected, active, err := release.Inspect(
			tx.Repo,
			"/run/current-system/etc/os-release",
		)
		if err != nil {
			return fail(errOut, err)
		}

		if expected != tx.ExpectedRelease {
			return fail(
				errOut,
				fmt.Errorf(
					"resume expected NixOS %s but repository policy now expects %s",
					tx.ExpectedRelease,
					expected,
				),
			)
		}

		if active != tx.ExpectedRelease {
			return fail(
				errOut,
				fmt.Errorf(
					"staged NixOS %s did not become active; current release is %s; automatic retry disabled",
					tx.ExpectedRelease,
					active,
				),
			)
		}

		fmt.Fprintf(
			out,
			"PASS: installer resumed on pinned NixOS %s\n",
			active,
		)

		args = append([]string(nil), tx.Args...)

		if err := os.Setenv("GJALLAR_INSTALLER_RESUMED", "1"); err != nil {
			return fail(
				errOut,
				fmt.Errorf(
					"mark validated installer resume transaction: %w",
					err,
				),
			)
		}
	}

	originalArgs := append([]string(nil), args...)

	f := flag.NewFlagSet("gjallar-installer", flag.ContinueOnError)
	f.SetOutput(errOut)
	cwd, _ := os.Getwd()
	opt := options{}
	f.StringVar(&opt.repo, "repo", cwd, "GjallarOS repository")
	f.BoolVar(&opt.skipHardware, "skip-hardware", false, "skip hardware generation")
	f.BoolVar(&opt.refreshHardware, "refresh-hardware", false, "regenerate hardware configuration")
	f.BoolVar(&opt.noRebuild, "no-rebuild", false, "do not install a boot generation")
	f.BoolVar(&opt.acceptExisting, "accept-existing", false, "allow an existing GjallarOS installation to be updated")
	f.BoolVar(&opt.forceRedeploy, "force-redeploy", false, "force a full clean GjallarOS redeployment without repartitioning or wiping user data")
	f.BoolVar(&opt.recovery, "recovery", false, "run from the trusted GjallarOS recovery environment")
	f.StringVar(&opt.targetDisk, "target-disk", "", "whole physical disk for a destructive fresh GjallarOS installation")
	f.StringVar(&opt.recoveryDisk, "recovery-disk", "", "GPT disk with unallocated space for a recovery partition")
	f.StringVar(&opt.recoveryPartition, "recovery-partition", "", "existing dedicated recovery partition")
	f.StringVar(&opt.recoverySigningKey, "recovery-signing-key", "", "runtime path to offline Ed25519 recovery release private key")
	f.StringVar(&opt.recoverySigningPublicKey, "recovery-signing-public-key", "", "runtime path to separately pinned Ed25519 recovery release public key")
	if err := f.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if f.NArg() != 0 {
		return 2
	}
	root, err := installercheck.ResolveRepository(opt.repo)
	if err != nil {
		return fail(errOut, err)
	}
	opt.repo = root
	if err := requireNixOS("/etc/os-release"); err != nil {
		return fail(errOut, err)
	}
	ui := prompt.New(in, out)
	s := state{control: controlBinary()}
	installedRoot := "/"
	presetRoot := root
	if opt.recovery && opt.acceptExisting {
		installedRoot = "/mnt"
		presetRoot = installedRoot
	}
	presetPath := filepath.Join(presetRoot, "user.config.json")
	if _, err := os.Stat(presetPath); err == nil {
		s.user, err = config.Load(presetPath)
		if err != nil {
			return fail(errOut, err)
		}
		s.preset = true
	}
	if opt.recovery {
		if opt.acceptExisting {
			s.existing, err = detectRecoveryInstalledRoot(ctx, installedRoot)
			if err != nil {
				return fail(errOut, err)
			}
		} else {
			// The embedded repository is installation source material, not evidence
			// that the recovery environment itself is an installed GjallarOS system.
			// Recovery without --accept-existing is the explicit fresh operation.
			s.existing = false
		}
	} else {
		s.existing, err = detectExistingInstalledSystem(ctx, root)
		if err != nil {
			return fail(errOut, err)
		}
	}

	persistentInstalledHost := s.existing
	if !opt.recovery && !persistentInstalledHost {
		persistentInstalledHost, err = detectPersistentInstalledHost(ctx)
		if err != nil {
			return fail(errOut, err)
		}
	}

	forceRedeploy := opt.forceRedeploy || s.user.ForceRedeploy

	if s.existing &&
		!forceRedeploy &&
		!opt.acceptExisting &&
		!s.user.UnattendedInstall {
		forceRedeploy, err = ui.Confirm(
			ctx,
			"Force a full clean GjallarOS redeployment instead of an in-place update? This reinstalls the system configuration without repartitioning, formatting, or wiping user data.",
			false,
		)
		if err != nil {
			return fail(errOut, err)
		}

	}

	opt.forceRedeploy = forceRedeploy

	if forceRedeploy && opt.noRebuild {
		return fail(
			errOut,
			errors.New("force redeployment requires a rebuild; --force-redeploy cannot be combined with --no-rebuild"),
		)
	}

	if s.existing && forceRedeploy {
		fmt.Fprintln(
			out,
			"Force clean redeployment selected; preserving disk layout, credentials, encryption keys, and user data.",
		)
	} else if s.existing && !opt.acceptExisting {
		approved, err := ui.Confirm(ctx, "Existing GjallarOS installation detected. Update it in place while preserving passwords, disk keys, and hardware configuration?", false)
		if err != nil {
			return fail(errOut, err)
		}
		if !approved {
			fmt.Fprintln(out, "Existing installation left unchanged.")
			return 0
		}
	}
	if opt.recovery {
		// Recovery media is a purpose-built execution environment. Never mutate
		// or rebuild the live recovery system before operating on the target.
		fmt.Fprintln(out, "Recovery environment validated; live-host rebuild skipped.")
	} else if code := prepareHost(
		ctx,
		ui,
		opt,
		s,
		originalArgs,
		out,
		errOut,
	); code != 0 {
		if code == installerReleaseRebootScheduled {
			return 0
		}
		return code
	}
	sourceRevision, err := sourcerevision.Resolve(root, opt.recovery)
	if err != nil {
		return fail(errOut, fmt.Errorf("resolve GjallarOS source revision: %w", err))
	}

	hardware := detectInstallerHardware("/sys")

	resolvedSource := currentODDCSource(root, sourceRevision)
	resolvedDevice, err := resolveODDCModelFromSource(
		resolvedSource,
		hardware,
	)
	if err != nil {
		return fail(errOut, err)
	}

	needsDeviceRebind := false
	if opt.recovery && opt.acceptExisting {
		capsulePath := filepath.Join(
			installedRoot,
			"var",
			"lib",
			"gjallarOS",
			"device-profile",
		)

		capsule, err := deviceprofilecache.Verify(capsulePath)
		if err != nil {
			return fail(
				errOut,
				fmt.Errorf("verify recovery device-profile capsule: %w", err),
			)
		}

		needsDeviceRebind = deviceprofilecache.NeedsRebind(
			capsule,
			discovery.ODDCIdentity(hardware),
			resolvedDevice,
		)

		if !needsDeviceRebind {
			resolvedSource = oddc.EmbeddedSource{
				Root:       filepath.Join(capsulePath, "oddc"),
				Repository: capsule.Source.ODDCRepository,
				Revision:   capsule.Source.ODDCRevision,
				Integrity:  capsule.Source.ODDCIntegrity,
			}

			resolvedDevice, err = resolveODDCModelFromSource(
				resolvedSource,
				hardware,
			)
			if err != nil {
				return fail(
					errOut,
					fmt.Errorf("resolve cached recovery device profile: %w", err),
				)
			}
		}
	}

	var activeHostOverlay *oddc.HostOverlay

	if shouldApplyODDCHostOverlay(
		persistentInstalledHost,
		needsDeviceRebind,
	) {
		resolvedDevice, activeHostOverlay, err = resolveODDCModelWithHost(
			ctx,
			installedRoot,
			resolvedSource,
			hardware,
			resolvedDevice,
		)
		if err != nil {
			return fail(errOut, err)
		}
	}

	hostOverlayChanged := false

	hardware, activeHostOverlay, hostOverlayChanged, err =
		reconcileInternalHardware(
			ctx,
			ui,
			resolvedDevice,
			hardware,
			activeHostOverlay,
			s.user.UnattendedInstall,
			out,
		)
	if err != nil {
		return fail(errOut, err)
	}

	if hostOverlayChanged {
		resolvedDevice, err = resolveODDCModelFromSourceWithHost(
			resolvedSource,
			hardware,
			activeHostOverlay,
		)
		if err != nil {
			return fail(
				errOut,
				fmt.Errorf(
					"apply staged machine-local ODDC hardware removals: %w",
					err,
				),
			)
		}
	}

	s.touchscreen = hardware.Touchscreen
	s.penTablet = hardware.PenTablet
	s.orientationSensor = hardware.OrientationSensor

	if _, err := oddc.ResolveGraphicsPolicy(resolvedDevice); err != nil {
		return fail(errOut, err)
	}

	fmt.Fprintf(out, "ODDC canonical model: %s\n", resolvedDevice.ModelID)

	if needsDeviceRebind {
		if s.user.UnattendedInstall || s.user.EndpointManagedDevice {
			return fail(
				errOut,
				errors.New(
					"cached recovery device profile does not match current hardware; explicit device rebind authorization is required",
				),
			)
		}

		approved, err := ui.Confirm(
			ctx,
			"Recovery device identity does not match the current hardware. Rebind recovery state to this machine and regenerate device-specific configuration?",
			false,
		)
		if err != nil {
			return fail(errOut, err)
		}
		if !approved {
			return fail(
				errOut,
				errors.New("device rebind was not authorized"),
			)
		}

		opt.refreshHardware = true
		fmt.Fprintln(out, "Authorized device rebind; hardware configuration will be regenerated.")
	}

	pinnedRelease, err := release.Expected(root)
	if err != nil {
		return fail(errOut, fmt.Errorf("resolve pinned NixOS release: %w", err))
	}

	if err := enforceDeviceValidation(
		ctx,
		ui,
		s.user,
		resolvedDevice,
		pinnedRelease,
		sourceRevision,
	); err != nil {
		return fail(errOut, err)
	}

	s.secureBootFirmware, err = oddc.ResolveSecureBootFirmwarePolicy(resolvedDevice)
	if err != nil {
		return fail(errOut, fmt.Errorf(
			"resolve Secure Boot firmware policy: %w",
			err,
		))
	}

	fmt.Fprintf(out, "Touchscreen detected: %t\n", hardware.Touchscreen)
	fmt.Fprintf(out, "Pen/tablet detected: %t\n", hardware.PenTablet)

	choices, err := discovery.Discover(root)
	if err != nil {
		return fail(errOut, err)
	}
	if s.preset {
		normalizePreset(&s.user, root)
	} else if err := collectInteractive(ctx, ui, root, hardware, choices, &s.user); err != nil {
		return fail(errOut, err)
	}
	if err := configureWeatherLocation(ctx, ui, &s.user, out); err != nil {
		return fail(errOut, err)
	}
	// Resolve managed security requirements before checking the physical TPM
	// boundary so policy normalization cannot re-enable TPM-dependent features
	// after a no-TPM decision.
	normalizeManagementSafety(&s.user)

	tpmAvailable := diskcrypto.TPMAvailable()
	securityRequested := s.user.SecureBootPrompt ||
		s.user.SecureBootEnable ||
		s.user.LUKSTPM2Enable ||
		s.user.JODSPrebootLockEnable

	if !tpmAvailable && securityRequested {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "TPM2 hardware was not detected.")
		fmt.Fprintln(
			out,
			"GjallarOS requires TPM2 for its Secure Boot and measured-boot security policy.",
		)
		fmt.Fprintln(
			out,
			"Secure Boot, TPM2 LUKS unlock, and TPM-dependent JODS preboot locking must be disabled to continue.",
		)

		if s.user.UnattendedInstall {
			return fail(
				errOut,
				errors.New(
					"TPM2 is unavailable while user.config.json requests TPM-dependent security; rerun interactively to approve disabling Secure Boot",
				),
			)
		}

		continued, err := ui.Confirm(
			ctx,
			"Continue installation with Secure Boot disabled?",
			false,
		)
		if err != nil {
			return fail(errOut, err)
		}
		if !continued {
			fmt.Fprintln(
				out,
				"Installation cancelled. Secure Boot settings were not changed.",
			)
			return 0
		}

		disableTPMDependentSecurity(&s.user)

		if s.preset {
			if err := config.WriteAtomic(presetPath, s.user); err != nil {
				return fail(
					errOut,
					fmt.Errorf(
						"persist no-TPM security settings to user.config.json: %w",
						err,
					),
				)
			}

			fmt.Fprintln(
				out,
				"Updated user.config.json for this TPM-less machine: secureBootPrompt=false, secureBootEnable=false, luksTpm2Enable=false, jodsPrebootLockEnable=false.",
			)
		}
	}

	if tpmAvailable &&
		s.user.SecureBootPrompt &&
		!s.user.EndpointManagedDevice &&
		s.secureBootFirmware.Policy.Supported {
		promptText := "Prepare Secure Boot and recovery keys?"
		if firmwareName := strings.TrimSpace(
			s.secureBootFirmware.Policy.FirmwareName,
		); firmwareName != "" {
			promptText = "Prepare Secure Boot and recovery keys for " +
				firmwareName + "?"
		}

		s.user.SecureBootEnable, err = ui.Confirm(
			ctx,
			promptText,
			false,
		)
		if err != nil {
			return fail(errOut, err)
		}
	}

	if err := validateSecureBootFirmwareSupport(
		s.user.SecureBootEnable,
		resolvedDevice.ModelID,
		s.secureBootFirmware,
	); err != nil {
		if mayOfferSecureBootFallback(
			s.user.SecureBootEnable,
			s.user.EndpointManagedDevice,
			s.user.UnattendedInstall,
		) {

			continued, promptErr := ui.Confirm(
				ctx,
				fmt.Sprintf(
					"%v Continue installation with Secure Boot disabled?",
					err,
				),
				false,
			)
			if promptErr != nil {
				return fail(errOut, promptErr)
			}
			if !continued {
				return fail(errOut, err)
			}

			disableTPMDependentSecurity(&s.user)

			if s.preset {
				if err := config.WriteAtomic(presetPath, s.user); err != nil {
					return fail(
						errOut,
						fmt.Errorf(
							"persist unsupported-firmware security fallback to user.config.json: %w",
							err,
						),
					)
				}

				fmt.Fprintln(
					out,
					"Updated user.config.json to keep unsupported Secure Boot and TPM2-dependent security disabled.",
				)
			}

			fmt.Fprintln(
				out,
				"Continuing installation with Secure Boot and TPM2 measured-boot unlock disabled.",
			)
		} else {
			return fail(errOut, err)
		}
	}

	if s.existing &&
		s.user.RecoveryEnable &&
		s.user.RecoveryPartitionEnable {
		continued, err := handleExistingRecoveryFilesystem(
			ctx,
			ui,
			presetPath,
			&s,
			out,
		)
		if err != nil {
			return fail(errOut, err)
		}
		if !continued {
			fmt.Fprintln(
				out,
				"Existing installation left unchanged; recovery compatibility was not accepted.",
			)
			return 0
		}
	}

	if err := configureRecoveryProvisioning(ctx, ui, opt, &s); err != nil {
		return fail(errOut, err)
	}
	if s.user.EndpointManagedDevice && s.user.JODSAllowInsecureTLS {
		if err := confirmInsecureJODS(ctx, ui); err != nil {
			return fail(errOut, err)
		}
	}
	// Management-only boot restrictions must never survive in an unmanaged
	// preset or a user declining JODS enrollment.
	if err := validateSelections(s.user, choices); err != nil {
		return fail(errOut, err)
	}
	if s.user.JODSPrebootLockEnable && (!s.user.RecoveryEnable || !s.user.SecureBootEnable) {
		return fail(errOut, errors.New("JODS preboot locking requires both the trusted recovery entry and Secure Boot"))
	}
	if err := config.Validate(s.user); err != nil {
		return fail(errOut, err)
	}
	if err := detectAndRenderState(
		ctx,
		root,
		&s,
		hardware,
		resolvedDevice,
	); err != nil {
		return fail(errOut, err)
	}
	if err := configureSecrets(ctx, root, &s, errOut); err != nil {
		return fail(errOut, err)
	}
	fmt.Fprintf(out, "\nSelected: hostname=%s user=%s shell=%s\n", s.user.Hostname, s.user.Username, s.user.Shell)
	write := s.user.WriteConfig
	if !s.preset {
		write, err = ui.Confirm(ctx, "Write configuration to "+filepath.Join(root, "generated", "state.nix")+"?", false)
		if err != nil {
			return fail(errOut, err)
		}
	}
	if !write {
		fmt.Fprintln(out, "Nothing changed.")
		return 0
	}
	if !s.preset {
		if err := config.WriteAtomic(presetPath, s.user); err != nil {
			return fail(errOut, err)
		}
		fmt.Fprintln(out, "Wrote", presetPath)
	}
	// A managed fresh installation may not create local account credentials
	// before the device has completed its JODS pre-install enrollment and
	// recovery-identity attestation. The legacy post-install enrollment path
	// is intentionally insufficient for this boundary.
	if s.render.EndpointManagedDevice && !s.existing {
		return fail(
			errOut,
			errors.New(
				"managed fresh installation requires JODS pre-install device enrollment and recovery-identity attestation before password provisioning",
			),
		)
	}

	rootPasswordPath := "/var/lib/gjallarOS/passwords/root.hash"
	if s.existing && privilegedFileExists(ctx, rootPasswordPath) {
		s.render.RootPasswordFile = rootPasswordPath
		fmt.Fprintln(out, "Existing root password hash retained.")
	} else {
		// Root recovery is intentionally local even on JODS-managed endpoints.
		// Management must never remove wheel/Polkit administration before a
		// verified local recovery credential exists.
		path, err := controlOutput(ctx, s.control, errOut, "installer", "local-password", "--username", "root", "--apply")
		if err != nil {
			return fail(errOut, err)
		}
		lines := strings.Fields(strings.TrimSpace(path))
		if len(lines) > 0 {
			s.render.RootPasswordFile = lines[len(lines)-1]
		}
		fmt.Fprint(out, path)
	}
	settingsPath := filepath.Join(root, "generated", "state.nix")
	if err := nixrender.WriteAtomic(settingsPath, s.render); err != nil {
		return fail(errOut, err)
	}
	fmt.Fprintln(out, "Wrote", settingsPath)
	hardwarePath := hardwareconfig.Target(root)
	hardwareGenerator := hardwareconfig.Generate
	if opt.recovery && opt.acceptExisting {
		hardwareGenerator = hardwareconfig.GenerateTarget
	}

	hardwareResult, err := reconcileHardwareConfiguration(
		ctx,
		root,
		hardwarePath,
		s.existing,
		opt.skipHardware,
		opt.refreshHardware,
		time.Now(),
		hardwareGenerator,
	)
	if err != nil {
		return fail(errOut, err)
	}

	skip := hardwareResult.action != hardwareGenerate

	switch hardwareResult.action {
	case hardwareSkip:
		fmt.Fprintln(out, "Hardware generation explicitly skipped.")

	case hardwareRetain:
		fmt.Fprintln(out, "Existing hardware configuration retained:", hardwarePath)

	case hardwareGenerate:
		if s.existing && !hardwareResult.existed {
			fmt.Fprintln(
				out,
				"Existing installation is missing hardware configuration; generating it from the running machine.",
			)
		} else if opt.refreshHardware {
			fmt.Fprintln(
				out,
				"Hardware refresh requested; regenerating from the running machine.",
			)
		}

		fmt.Fprintln(out, "PLAN: generate and atomically replace", hardwarePath)
		if hardwareResult.backup != "" {
			fmt.Fprintln(out, "Backup:", hardwareResult.backup)
		}
		if !tpm2AllowedForSecureBoot(s.render.SecureBootEnable) {
			s.render.LUKSTPM2Enable = false
			s.user.LUKSTPM2Enable = false
			fmt.Fprintln(
				out,
				"Secure Boot is disabled; TPM2 measured-boot unlock remains disabled.",
			)
		} else if s.render.EndpointManagedDevice {
			s.render.LUKSTPM2Enable = s.user.LUKSTPM2Enable
		} else {
			tpmOut, err := controlOutput(ctx, s.control, errOut, "installer", "tpm2", "--repo", root, "--hardware", hardwarePath)
			if err != nil {
				return fail(errOut, err)
			}
			fmt.Fprint(out, tpmOut)
			s.render.LUKSTPM2Enable = strings.Contains(tpmOut, "luks_tpm2_enable=true")
		}
		if err := nixrender.WriteAtomic(settingsPath, s.render); err != nil {
			return fail(errOut, err)
		}
		if !s.render.EndpointManagedDevice && !s.existing {
			// A fresh target does not have its canonical LUKS2 root yet.
			// Running installer luks here would operate against the installer
			// environment instead of the future target.
			//
			// rootprovision creates the initial human recovery-capable keyslot.
			// TPM2 enrollment is intentionally deferred until the installed
			// GjallarOS lifecycle can act against its own verified root device.
			fmt.Fprintln(
				out,
				"Fresh-target TPM2 LUKS enrollment deferred until the installed GjallarOS lifecycle.",
			)
		} else if s.existing {
			fmt.Fprintln(out, "Existing LUKS keyslots retained; disk-key migration was not rerun.")
		}
	}
	if s.render.JODSPrebootLockEnable && !s.render.LUKSTPM2Enable {
		return fail(errOut, errors.New("JODS preboot locking requires verified TPM2 LUKS enrollment; configuration was left unactivated"))
	}

	if opt.recovery {
		// Recovery media uses an embedded writable source tree, not a Git
		// checkout. Local Git excludes are installed/development-tree hygiene
		// and are not applicable to the recovery execution environment.
		fmt.Fprintln(out, "Recovery installer source is embedded; local Git protection skipped.")
	} else {
		fmt.Fprintln(out, "PLAN: atomically update local Git excludes and protect generated machine configuration")
		if _, err := localgit.Protect(ctx, root, func() string {
			if skip {
				return ""
			}
			return hardwarePath
		}()); err != nil {
			return fail(errOut, err)
		}
	}
	if s.render.SecureBootEnable {
		inspection, err := secureboot.Inspect(ctx)
		if err != nil {
			return fail(errOut, fmt.Errorf("inspect existing Secure Boot ownership: %w", err))
		}
		fmt.Fprintf(out, "Secure Boot state: %s -- %s\n", inspection.State, inspection.Description)

		switch inspection.State {
		case secureboot.StateGjallarManaged:
			fmt.Fprintln(out, "Existing GjallarOS Secure Boot ownership detected; reusing the current PK/KEK/db key set.")
			if !inspection.Recorded {
				if err := secureboot.RecordOwnership(ctx, "enrolled"); err != nil {
					return fail(errOut, err)
				}
				fmt.Fprintln(out, "Recorded legacy GjallarOS/sbctl Secure Boot ownership without rotating keys.")
			}

		case secureboot.StatePendingEnrollment:
			fmt.Fprintln(out, "Existing GjallarOS Secure Boot provisioning is pending; reusing the prepared key set.")

		case secureboot.StateOEMFactoryDerived:
			if inspection.SecureBoot {
				return fail(errOut, errors.New("factory Secure Boot is currently enforced; disable Secure Boot before transferring ownership to GjallarOS"))
			}

			fmt.Fprintln(out, "OEM/factory-derived Secure Boot root detected; preparing machine-specific GjallarOS ownership.")

			// Provision() creates a root-only temporary recovery
			// checkpoint so an interrupted installer can resume with the
			// same recovery archive/passphrase.
			recovery, err := secureboot.Provision(ctx)
			if err != nil {
				return fail(errOut, err)
			}
			recovery.Passphrase = ""

			if err := secureboot.RecordOwnership(ctx, "pending-enrollment"); err != nil {
				return fail(errOut, err)
			}

		case secureboot.StateSetupModeUnknown:
			return fail(errOut, errors.New("firmware is already in Secure Boot Setup Mode but GjallarOS cannot prove that it initiated the transition; refusing automatic enrollment"))

		case secureboot.StateForeignManaged:
			return fail(errOut, errors.New("an existing non-GjallarOS Secure Boot owner configuration was detected; refusing to replace it automatically"))

		case secureboot.StateInconsistent:
			return fail(errOut, errors.New("GjallarOS Secure Boot ownership metadata does not match the current key material; refusing to continue"))

		default:
			return fail(errOut, fmt.Errorf("unsupported Secure Boot ownership state %q", inspection.State))
		}
	}
	hostOverlayToCommit, err := hostOverlayForCommit(
		activeHostOverlay,
		hostOverlayChanged,
		needsDeviceRebind,
		resolvedDevice.ModelID,
	)
	if err != nil {
		return fail(errOut, err)
	}

	runRebuild := s.user.RunRebuild && !opt.noRebuild
	if opt.forceRedeploy {
		runRebuild = true
	}
	if !s.preset && !opt.noRebuild && !opt.forceRedeploy {
		runRebuild, err = ui.Confirm(ctx, "Install the next NixOS boot generation now?", false)
		if err != nil {
			return fail(errOut, err)
		}
	}
	if needsDeviceRebind && !runRebuild {
		return fail(
			errOut,
			errors.New(
				"Recovery device rebind requires installing the regenerated system; rebuild cannot be skipped",
			),
		)
	}

	if runRebuild {
		if persistentInstalledHost {
			maintenanceRequired := false

			if s.user.RecoveryEnable &&
				s.user.RecoveryPartitionEnable &&
				s.recoveryPartition == "" {
				fmt.Fprintln(
					out,
					"STAGE: inspecting existing encrypted-Btrfs layout for recovery storage",
				)

				maintenanceRequired, err = prepareInstalledRecoveryStorage(
					ctx,
					ui,
					out,
					installedRoot,
					&s,
				)
				if err != nil {
					return fail(errOut, err)
				}

				if maintenanceRequired {
					s.recoveryPartition = ""
				}
			}

			if opt.recovery && opt.acceptExisting {
				fmt.Fprintln(
					out,
					"STAGE: preparing authenticated recovery target",
				)

				if err := recoverytarget.PrepareBoot(
					ctx,
					installedRoot,
				); err != nil {
					return fail(errOut, err)
				}

				fmt.Fprintln(
					out,
					"PLAN: reinstall existing system into",
					installedRoot,
				)

				if _, err := baremetalinstall.Install(
					ctx,
					baremetalinstall.Input{
						Repo:     root,
						Hostname: s.user.Hostname,
						Out:      out,
					},
				); err != nil {
					return fail(errOut, err)
				}

				if needsDeviceRebind {
					if err := materializeODDCCapsule(
						root,
						filepath.Join(
							installedRoot,
							"var",
							"lib",
							"gjallarOS",
							"device-profile",
						),
						hardware,
						resolvedDevice,
						true,
						"hardware-rebind",
					); err != nil {
						return fail(
							errOut,
							fmt.Errorf(
								"commit recovery device-profile capsule after hardware rebind: %w",
								err,
							),
						)
					}

					fmt.Fprintln(
						out,
						"PASS: recovery device profile rebound to current hardware.",
					)
				}

				if err := saveODDCHostOverlay(
					ctx,
					installedRoot,
					hostOverlayToCommit,
				); err != nil {
					return fail(errOut, err)
				}

				if hostOverlayToCommit != nil {
					fmt.Fprintln(
						out,
						"PASS: machine-local ODDC host state committed after recovery deployment.",
					)
				}
			} else {
				target, err := deploy.Target(
					root,
					s.user.Hostname,
				)
				if err != nil {
					return fail(errOut, err)
				}

				fmt.Fprintln(
					out,
					"PLAN: validate then install",
					target,
				)

				if err := deploy.Apply(ctx, target); err != nil {
					return fail(errOut, err)
				}

				if err := saveODDCHostOverlay(
					ctx,
					installedRoot,
					hostOverlayToCommit,
				); err != nil {
					return fail(errOut, err)
				}

				if hostOverlayToCommit != nil {
					fmt.Fprintln(
						out,
						"PASS: machine-local ODDC host state committed after deployment.",
					)
				}
			}

			if maintenanceRequired {
				fmt.Fprintln(
					out,
					"PASS: one-shot recovery-storage maintenance environment installed",
				)
				fmt.Fprintln(
					out,
					"STAGE: arming one-shot maintenance boot",
				)

				systemPath, err := filepath.EvalSymlinks(
					filepath.Join(installedRoot, "nix/var/nix/profiles/system"),
				)
				if err != nil {
					return fail(
						errOut,
						fmt.Errorf(
							"resolve installed system generation: %w",
							err,
						),
					)
				}

				maintenanceNext := filepath.Join(
					systemPath,
					"sw/bin/gjallar-recovery-maintenance-next",
				)

				if err := attached(
					ctx,
					"sudo",
					maintenanceNext,
					installedRoot,
				); err != nil {
					return fail(
						errOut,
						fmt.Errorf(
							"arm recovery-storage maintenance boot: %w",
							err,
						),
					)
				}

				return 0
			}
			if err := provisionRecoveryPartition(ctx, root, s); err != nil {
				return fail(errOut, err)
			}
		} else {
			targetDisk, err := selectFreshTargetDisk(
				ctx,
				ui,
				out,
				opt.targetDisk,
			)
			if err != nil {
				return fail(
					errOut,
					fmt.Errorf(
						"select fresh installation target: %w",
						err,
					),
				)
			}

			result, err := runFreshBareMetal(
				ctx,
				ui,
				root,
				targetDisk,
				s.user.Hostname,
				hardware,
				resolvedDevice,
				opt.recovery,
				s.user.RecoveryEnable,
				[]string{s.render.RootPasswordFile},
				hostOverlayToCommit,
				out,
			)
			if err != nil {
				return fail(
					errOut,
					fmt.Errorf(
						"run canonical fresh bare-metal installation: %w",
						err,
					),
				)
			}

			s.recoveryPartition = result.RecoveryPartition
		}
	}

	// Secure Boot continuation is intentionally independent of runRebuild.
	//
	// -no-rebuild means "do not install another NixOS generation"; it must
	// never mean "skip an already-started Secure Boot transaction".
	if s.render.SecureBootEnable {
		inspection, err := secureboot.Inspect(ctx)
		if err != nil {
			return fail(
				errOut,
				fmt.Errorf(
					"inspect Secure Boot state before continuation: %w",
					err,
				),
			)
		}

		needsRecoveryGate :=
			inspection.State == secureboot.StateOEMFactoryDerived ||
				inspection.State == secureboot.StatePendingEnrollment ||
				inspection.State == secureboot.StateGjallarManaged

		if needsRecoveryGate {
			confirmed, err := secureboot.RecoveryConfirmed(ctx)
			if err != nil {
				return fail(errOut, err)
			}

			if confirmed {
				fmt.Fprintln(
					out,
					"Secure Boot recovery material was already confirmed saved; resuming provisioning.",
				)
			} else {
				recovery, err := secureboot.Provision(ctx)
				if err != nil {
					return fail(
						errOut,
						fmt.Errorf(
							"prepare Secure Boot recovery material: %w",
							err,
						),
					)
				}

				for {
					if err := ui.ShowSecureBootRecovery(
						ctx,
						recovery.ArchivePath,
						recovery.Passphrase,
					); err != nil {
						recovery.Passphrase = ""
						return fail(
							errOut,
							fmt.Errorf(
								"display Secure Boot recovery material: %w",
								err,
							),
						)
					}

					saved, err := ui.Confirm(
						ctx,
						"Have you saved BOTH the Secure Boot recovery archive and passphrase?",
						false,
					)
					if err != nil {
						recovery.Passphrase = ""
						return fail(errOut, err)
					}

					if !saved {
						fmt.Fprintln(
							out,
							"Recovery material not confirmed saved; showing it again.",
						)
						continue
					}

					savedAgain, err := ui.Confirm(
						ctx,
						"Are you absolutely sure the Secure Boot recovery material is saved offline?",
						false,
					)
					if err != nil {
						recovery.Passphrase = ""
						return fail(errOut, err)
					}

					if !savedAgain {
						fmt.Fprintln(
							out,
							"Second confirmation declined; showing the recovery material again.",
						)
						continue
					}

					break
				}

				recovery.Passphrase = ""

				if err := secureboot.MarkRecoveryConfirmed(ctx); err != nil {
					return fail(errOut, err)
				}

				fmt.Fprintln(
					out,
					"Secure Boot recovery checkpoint completed; future interrupted runs resume after this gate.",
				)
			}
		}

		firmwarePolicy, err := secureboot.SnapshotFirmwarePolicy(
			resolvedDevice.ModelID,
			s.secureBootFirmware,
		)
		if err != nil {
			return fail(
				errOut,
				fmt.Errorf(
					"prepare detected Secure Boot firmware policy: %w",
					err,
				),
			)
		}

		next, err := secureboot.VerifyAndArmEnrollment(
			ctx,
			firmwarePolicy,
		)
		if err != nil {
			return fail(errOut, err)
		}

		switch next {
		case secureboot.ContinuationNone:
			fmt.Fprintln(
				out,
				"Secure Boot is active and the GjallarOS boot chain is verified.",
			)
			// Secure Boot is one installer stage, not the definition of
			// installation completion. Continue into the normal final
			// completion path below.

		case secureboot.ContinuationEnable:
			fmt.Fprintln(
				out,
				"GjallarOS Secure Boot ownership is already enrolled.",
			)
			fmt.Fprintln(
				out,
				"Only Secure Boot enforcement still needs to be enabled in firmware.",
			)

			if err := ui.SecureBootEnableHandoff(
				ctx,
				firmwarePolicy.FirmwareName,
			); err != nil {
				return fail(
					errOut,
					fmt.Errorf("Secure Boot enable handoff: %w", err),
				)
			}

		case secureboot.ContinuationEnroll:
			fmt.Fprintln(
				out,
				"Secure Boot ownership transfer armed and boot artifacts verified.",
			)

			if err := ui.SecureBootFirmwareHandoff(
				ctx,
				firmwarePolicy.FirmwareName,
				firmwarePolicy.Instructions,
			); err != nil {
				return fail(
					errOut,
					fmt.Errorf("Secure Boot firmware handoff: %w", err),
				)
			}

		default:
			return fail(
				errOut,
				fmt.Errorf(
					"unknown Secure Boot continuation state %q",
					next,
				),
			)
		}

		// A completed Secure Boot transaction needs no marker, firmware
		// handoff, or reboot. This is especially important on installer reruns.
		if secureBootNeedsFirmwareReboot(next) {
			if err := attached(
				ctx,
				"sudo", "install",
				"-d", "-m", "0700",
				"/var/lib/gjallarOS",
			); err != nil {
				return fail(
					errOut,
					fmt.Errorf(
						"create installer continuation directory: %w",
						err,
					),
				)
			}

			if err := attached(
				ctx,
				"sudo", "install",
				"-m", "0600",
				"/dev/null",
				installerSecureBootResumeMarker,
			); err != nil {
				return fail(
					errOut,
					fmt.Errorf(
						"arm installer continuation across Secure Boot reboot: %w",
						err,
					),
				)
			}

			fmt.Fprintln(
				out,
				"Installer continuation armed across Secure Boot reboot.",
			)
			fmt.Fprintln(
				out,
				"After Secure Boot is verified, GjallarOS will automatically run the final installer checks.",
			)

			fmt.Fprintln(
				out,
				"Secure Boot firmware instructions acknowledged.",
			)
			fmt.Fprintln(
				out,
				"Rebooting directly into firmware setup...",
			)

			if err := attached(
				ctx,
				"sudo", "systemctl",
				"reboot", "--firmware-setup",
			); err != nil {
				return fail(
					errOut,
					fmt.Errorf("reboot into firmware setup: %w", err),
				)
			}

			return 0
		}
	}

	if runRebuild {
		if s.render.EndpointManagedDevice {
			if s.existing {
				if err := activateJODSEnrollment(ctx, true); err != nil {
					return fail(errOut, err)
				}
				fmt.Fprintln(out, "JODS enrollment submitted; retry timer enabled while approval is pending.")
			} else {
				fmt.Fprintln(
					out,
					"JODS enrollment deferred to the installed GjallarOS lifecycle; installer environment was not enrolled.",
				)
			}
		}
		if s.user.AutoReboot {
			_ = attached(ctx, "sudo", "systemctl", "reboot")
		} else if !s.render.EndpointManagedDevice {
			yes, _ := ui.Confirm(
				ctx,
				"Deployment complete. Reboot now?",
				false,
			)
			if yes {
				_ = attached(ctx, "sudo", "systemctl", "reboot")
			}
		}
	}
	if s.render.EndpointManagedDevice && !runRebuild {
		fmt.Fprintln(out, "JODS configured, not contacted: enrollment waits for a successful installation rebuild.")
	}

	fmt.Fprintln(out, "GjallarOS installation complete.")
	return 0
}

func confirmInsecureJODS(ctx context.Context, ui prompt.UI) error {
	confirmed, err := ui.Confirm(ctx, "SECURITY WARNING: disable TLS certificate verification for this local-development JODS endpoint? This permits machine-in-the-middle attacks.", false)
	if err != nil {
		return err
	}
	if !confirmed {
		return errors.New("insecure JODS TLS was not explicitly confirmed")
	}
	return nil
}

func activateJODSEnrollment(ctx context.Context, installationSucceeded bool) error {
	if !installationSucceeded {
		return nil
	}
	if err := runJODSCommand(ctx, "sudo", "install", "-d", "-m", "0700", "/var/lib/gjallarOS"); err != nil {
		return fmt.Errorf("prepare installer status directory: %w", err)
	}
	if err := runJODSCommand(ctx, "sudo", "install", "-m", "0600", "/dev/null", "/var/lib/gjallarOS/installation-complete"); err != nil {
		return fmt.Errorf("mark installation complete: %w", err)
	}
	if err := runJODSCommand(ctx, "sudo", "systemctl", "start", "jods-mdm-agent-enroll.service"); err != nil {
		return fmt.Errorf("submit JODS enrollment: %w", err)
	}
	if err := runJODSCommand(ctx, "sudo", "systemctl", "enable", "--now", "jods-mdm-agent-enroll.timer"); err != nil {
		return fmt.Errorf("enable JODS enrollment retry timer: %w", err)
	}
	return nil
}

func prepareHost(
	ctx context.Context,
	ui prompt.UI,
	opt options,
	s state,
	originalArgs []string,
	out,
	errOut io.Writer,
) int {
	expected, actual, err := release.Inspect(opt.repo, "/etc/os-release")
	if err != nil {
		return fail(errOut, err)
	}
	if expected != actual {
		fmt.Fprintf(out, "NixOS %s detected; target is %s.\n", actual, expected)
		fmt.Fprintf(
			out,
			"ACTION: staging pinned NixOS %s for the next boot; the live desktop will not be switched in place.\n",
			expected,
		)

		executable, err := os.Executable()
		if err != nil {
			return fail(
				errOut,
				fmt.Errorf(
					"resolve installer executable for release continuation: %w",
					err,
				),
			)
		}

		if err := installerresume.Arm(
			ctx,
			executable,
			opt.repo,
			originalArgs,
			expected,
		); err != nil {
			return fail(
				errOut,
				fmt.Errorf(
					"arm installer release continuation: %w",
					err,
				),
			)
		}

		if err := release.Align(
			ctx,
			expected,
			installerresume.WrapperPath,
		); err != nil {
			return fail(errOut, err)
		}

		fmt.Fprintf(
			out,
			"PASS: NixOS %s staged and installer continuation armed.\n",
			expected,
		)
		fmt.Fprintln(
			out,
			"Rebooting into the staged release; installation will resume automatically.",
		)

		if err := attached(
			ctx,
			"sudo",
			"systemctl",
			"reboot",
		); err != nil {
			return fail(
				errOut,
				fmt.Errorf(
					"reboot into staged NixOS %s: %w; continuation remains armed for the next manual reboot",
					expected,
					err,
				),
			)
		}

		return installerReleaseRebootScheduled
	}
	// A completed GjallarOS installation is flake-owned. Rebuilding the
	// bootstrap /etc/nixos/configuration.nix on a rerun would switch the live
	// machine back to its pre-install base generation before deploying the
	// requested flake generation.
	if s.existing && !opt.forceRedeploy {
		fmt.Fprintln(out, "Existing GjallarOS installation detected; skipping /etc/nixos bootstrap rebuild.")
	} else {
		if s.existing && opt.forceRedeploy {
			fmt.Fprintln(out, "Force redeployment requested; running prerequisite bootstrap despite existing-install detection.")
		}
		configPath := "/etc/nixos/configuration.nix"
		data, err := os.ReadFile(configPath)
		if err != nil {
			return fail(errOut, err)
		}
		plan := bootstrap.Build(data, s.preset)
		if len(plan.Missing) > 0 {
			fmt.Fprintf(out, "Missing helpers/options: %s\nProposed configuration:\n%s\n", strings.Join(plan.Missing, ", "), plan.Updated)
			if !s.user.EndpointManagedDevice && os.Getenv("GJALLAR_INSTALLER_RESUMED") != "1" {
				yes, err := ui.Confirm(ctx, "Apply prerequisite configuration and rebuild?", false)
				if err != nil {
					return fail(errOut, err)
				}
				if !yes {
					return fail(errOut, errors.New("required persistent prerequisites were not approved"))
				}
			}
			if _, err := bootstrap.Apply(ctx, configPath, plan.Updated, time.Now()); err != nil {
				return fail(errOut, err)
			}
		}
	}
	check := s.user.RunUpdateChecks
	if !s.preset && os.Getenv("GJALLAR_INSTALLER_RESUMED") != "1" {
		check, err = ui.Confirm(ctx, "Check for and install firmware updates?", false)
		if err != nil {
			return fail(errOut, err)
		}
	}
	if check && firmware.Available() {
		fmt.Fprintln(out, "PLAN: refresh and apply firmware updates")
		_, _ = firmware.Update(ctx)
	}
	return 0
}

func detectedNixSystem() (string, error) {
	switch runtime.GOARCH {
	case "amd64":
		return "x86_64-linux", nil
	case "arm64":
		return "aarch64-linux", nil
	default:
		return "", fmt.Errorf(
			"unsupported machine architecture %q",
			runtime.GOARCH,
		)
	}
}

func containsValue(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func collectInteractive(ctx context.Context, ui prompt.UI, root string, hardware discovery.Hardware, o discovery.Options, u *config.User) error {
	var err error
	host, _ := os.Hostname()
	if host == "" || host == "nixos" {
		host = "gjallarOS"
	}
	account, _ := user.Current()
	username := "user"
	if account != nil {
		username = account.Username
	}
	if u.Hostname, err = ui.Value(ctx, "Hostname", host); err != nil {
		return err
	}
	if u.Username, err = ui.Value(ctx, "Username", username); err != nil {
		return err
	}
	u.EndpointManagedDevice, err = ui.Confirm(ctx, "Manage this machine with JODS?", false)
	if err != nil {
		return err
	}
	if u.EndpointManagedDevice {
		localDevelopment, err := ui.Confirm(ctx, "Use a clearly marked local-development JODS server?", false)
		if err != nil {
			return err
		}
		endpointDefault := strings.TrimSpace(u.JODSEndpoint)
		if endpointDefault == "" && !localDevelopment {
			endpointDefault = "https://admin.oss-ad.eu:1666"
		}
		u.JODSEndpoint, err = ui.Value(ctx, "JODS HTTPS endpoint", endpointDefault)
		if err != nil {
			return err
		}
		u.JODSPolicySigningKey, err = ui.Value(ctx, "JODS policy-signing Ed25519 public key (64 hexadecimal characters)", "")
		if err != nil {
			return err
		}
		u.JODSRecoverySigningKey, err = ui.Value(ctx, "JODS recovery-command Ed25519 public key (different 64 hexadecimal characters)", "")
		if err != nil {
			return err
		}
		u.JODSEnrollmentMode, err = ui.Choice(ctx, "JODS enrollment mode", "manual", []string{"auto", "manual", "jade-registry-only"})
		if err != nil {
			return err
		}
		if localDevelopment {
			u.JODSAllowInsecureTLS, err = ui.Confirm(ctx, "Allow a self-signed JODS TLS certificate for local development?", false)
			if err != nil {
				return err
			}
		}
		deviceDefault := "pc"
		if hardware.FormFactor == "laptop" {
			deviceDefault = "laptop"
		}
		u.JODSDeviceClass, err = ui.Choice(ctx, "JODS device class", deviceDefault, []string{"pc", "vm", "laptop", "kiosk", "workstation"})
		if err != nil {
			return err
		}
		u.JODSDesktopProfile, err = ui.Value(ctx, "JODS desktop profile", "hyprland")
		if err != nil {
			return err
		}
		u.JODSDesktopProfile = strings.ToLower(strings.TrimSpace(u.JODSDesktopProfile))
	}
	if err != nil {
		return err
	}
	u.Timezone, err = ui.Value(ctx, "Timezone (IANA name)", "Europe/Berlin")
	if err != nil {
		return err
	}
	u.Locale, err = ui.Value(ctx, "Locale", "en_US.UTF-8")
	if err != nil {
		return err
	}
	layout, err := ui.Value(ctx, "Keyboard layout", "us")
	if err != nil {
		return err
	}
	normalized, err := xkb.Normalize(layout)
	if err != nil {
		return err
	}
	u.KeyboardLayout, u.KeyboardVariant = normalized.Name, normalized.Variant
	u.TouchpadWorkspaceSwipe, err = ui.Confirm(ctx, "Enable three-finger workspace swipes?", true)
	if err != nil {
		return err
	}
	u.ClamshellEnable, err = ui.Confirm(ctx, "Enable dock-aware clamshell mode?", true)
	if err != nil {
		return err
	}
	u.USBGuardEnable, err = ui.Confirm(ctx, "Enable USB trust review in audit mode? Blocking requires separate activation after device enrollment.", false)
	if err != nil {
		return err
	}
	u.Name, err = ui.Value(ctx, "Full name", u.Username)
	if err != nil {
		return err
	}
	u.Email, err = ui.Value(ctx, "Git email", u.Username+"@example.com")
	if err != nil {
		return err
	}
	u.GitHubUsername, err = ui.Value(ctx, "GitHub username", "")
	if err != nil {
		return err
	}
	u.DotfilesDir, err = ui.Value(ctx, "Absolute dotfiles path", root)
	if err != nil {
		return err
	}
	u.Shell, err = ui.Choice(ctx, "Login shell", first(o.Shells), o.Shells)
	if err != nil {
		return err
	}
	u.Editors, err = ui.Multi(ctx, "Editors", []string{"vscodium"}, o.Editors)
	if err != nil {
		return err
	}
	u.Browsers, err = ui.Multi(ctx, "Browsers", []string{first(o.Browsers)}, o.Browsers)
	if err != nil {
		return err
	}
	u.PreferredEditor, err = ui.Value(ctx, "Preferred editor", u.Editors[0])
	if err != nil {
		return err
	}
	u.PreferredBrowser, err = ui.Value(ctx, "Preferred browser", u.Browsers[0])
	if err != nil {
		return err
	}
	if err := collectProjectTools(ctx, ui, u); err != nil {
		return err
	}
	// Theme selection belongs to the desktop shell after installation.
	// Keep the default theme only as the bootstrap rendering contract.
	if u.Theme == "" {
		u.Theme = first(o.Themes)
	}
	u.PrintingEnable, err = ui.Confirm(ctx, "Enable printing and scanning support?", false)
	if err != nil {
		return err
	}
	u.NetworkPrintingEnable = false
	if u.PrintingEnable {
		u.NetworkPrintingEnable, err = ui.Confirm(ctx, "Enable network printer discovery?", false)
		if err != nil {
			return err
		}
	}
	u.ContainersEnable, err = ui.Confirm(ctx, "Enable rootless container tooling with Docker-compatible commands?", false)
	if err != nil {
		return err
	}
	u.AIEnable, err = ui.Confirm(ctx, "Enable local AI tools?", false)
	if err != nil {
		return err
	}
	if u.AIEnable {
		u.OverrideAISelection, err = ui.Confirm(ctx, "Override automatic hardware-aware AI model selection?", false)
		if err != nil {
			return err
		}
		if u.OverrideAISelection {
			u.OverrideModelWith, err = ui.Value(ctx, "Exact Ollama model identifier", "qwen2.5-coder:14b")
			if err != nil {
				return err
			}
		}
		u.AIAgentMode, err = ui.Choice(ctx, "Local AI permission profile", "workspace", []string{"workspace", "owner-conservative", "owner-full-local"})
		if err != nil {
			return err
		}
	}
	u.NemuEnable, err = ui.Confirm(ctx, "Enable Nemu virtual machines?", false)
	if err != nil {
		return err
	}
	u.RecoveryEnable, err = ui.Confirm(ctx, "Install the trusted local GjallarOS recovery/JODS boot entry?", true)
	if err != nil {
		return err
	}
	if u.RecoveryEnable {
		u.RecoveryPartitionEnable, err = ui.Confirm(
			ctx,
			"Create and maintain a dedicated GjallarOS recovery partition (recommended):",
			true,
		)
		if err != nil {
			return err
		}
	} else {
		u.RecoveryPartitionEnable = false
	}
	if u.RecoveryEnable && u.EndpointManagedDevice {
		u.JODSPrebootLockEnable, err = ui.Confirm(ctx, "Require Secure Boot and measured-boot TPM policy for JODS preboot access?", true)
		if err != nil {
			return err
		}
	}
	// Device-specific hardware policy is resolved through ODDC.
	// Do not ask users to manually select a vendor/model that hardware
	// discovery and the resolved ODDC device graph already determine.
	u.WriteConfig = true
	u.RunRebuild = true
	return nil
}

func collectProjectTools(ctx context.Context, ui prompt.UI, u *config.User) error {
	// Canonical installer intent is the generic webApplications list.
	// Legacy Plane/Draw.io fields are populated by normalization only as a
	// temporary compatibility bridge for older generated-state consumers.
	u.WebApplications = nil
	u.PlaneEnable = false
	u.PlaneHost = ""
	u.DrawioEnable = false
	u.DrawioSelfHosted = false
	u.DrawioHost = ""

	for _, id := range config.SupportedWebApplicationIDs() {
		label := id

		switch id {
		case config.WebApplicationPlane:
			label = "Plane"
		case config.WebApplicationDrawio:
			label = "Draw.io"
		case config.WebApplicationTeams:
			label = "Microsoft Teams"
		}

		enabled, err := ui.Confirm(
			ctx,
			"Enable "+label+" web application integration?",
			false,
		)
		if err != nil {
			return err
		}
		if !enabled {
			continue
		}

		application := config.WebApplicationIntent{ID: id}

		switch id {
		case config.WebApplicationPlane:
			application.Endpoint, err = ui.Value(
				ctx,
				"Plane endpoint",
				"",
			)
			if err != nil {
				return err
			}

		case config.WebApplicationDrawio:
			selfHosted, err := ui.Confirm(
				ctx,
				"Use a self-hosted Draw.io server?",
				false,
			)
			if err != nil {
				return err
			}

			if selfHosted {
				application.Endpoint, err = ui.Value(
					ctx,
					"Draw.io endpoint",
					"",
				)
				if err != nil {
					return err
				}
			}

		case config.WebApplicationTeams:
			// Teams owns its default service endpoint.

		default:
			return fmt.Errorf(
				"unsupported web application integration: %q",
				id,
			)
		}

		u.WebApplications = append(
			u.WebApplications,
			application,
		)
	}

	var err error

	u.NextcloudEnable, err = ui.Confirm(
		ctx,
		"Enable Nextcloud integration?",
		false,
	)
	if err != nil {
		return err
	}

	if u.NextcloudEnable {
		u.NextcloudHost, err = ui.Value(
			ctx,
			"Nextcloud server",
			"",
		)
		if err != nil {
			return err
		}
	}

	return config.NormalizeProjectTools(u)
}

func normalizePreset(u *config.User, root string) {
	if u.DotfilesDir == "" {
		u.DotfilesDir = filepath.Join("/home", u.Username, "Documents", "gjallarOS")
	}
	u.DotfilesDir = strings.ReplaceAll(u.DotfilesDir, "usernamehere", u.Username)
	if normalized, err := xkb.Normalize(u.KeyboardLayout); err == nil {
		u.KeyboardLayout, u.KeyboardVariant = normalized.Name, normalized.Variant
	}
}

func disableTPMDependentSecurity(u *config.User) {
	u.SecureBootPrompt = false
	u.SecureBootEnable = false
	u.LUKSTPM2Enable = false
	u.JODSPrebootLockEnable = false
}

func normalizeManagementSafety(u *config.User) {
	if u.EndpointManagedDevice {
		// A managed endpoint must never enter the managed state with a weaker
		// local boot/recovery posture. These are one security contract.
		u.SecureBootEnable = true
		u.LUKSTPM2Enable = true
		u.RecoveryEnable = true
		u.RecoveryPartitionEnable = true
		u.JODSPrebootLockEnable = true
		return
	}

	u.JODSPrebootLockEnable = false
}

func configureWeatherLocation(ctx context.Context, ui prompt.UI, u *config.User, out io.Writer) error {
	configuredCity := strings.TrimSpace(u.WeatherCity)
	configuredCountry := strings.TrimSpace(u.WeatherCountry)
	if configuredCity != "" && configuredCountry != "" {
		u.WeatherCity = configuredCity
		u.WeatherCountry = configuredCountry
		fmt.Fprintf(out, "Weather location configured: %s, %s\n", configuredCity, configuredCountry)
		return nil
	}

	detected, detectErr := detectNetworkLocation(ctx)
	if detectErr == nil {
		fmt.Fprintf(out, "Approximate network location detected: %s, %s\n", detected.City, detected.Country)
		correct, err := ui.Confirm(ctx, "Is this the correct weather location: "+detected.City+", "+detected.Country+"?", true)
		if err != nil {
			return err
		}
		if correct {
			u.WeatherCity, u.WeatherCountry = detected.City, detected.Country
			return nil
		}
	} else {
		fmt.Fprintf(out, "Network location detection unavailable: %v\n", detectErr)
	}

	city, err := ui.Value(ctx, "Weather city", u.WeatherCity)
	if err != nil {
		return err
	}
	country, err := ui.Value(ctx, "Weather country", u.WeatherCountry)
	if err != nil {
		return err
	}
	city, country = strings.TrimSpace(city), strings.TrimSpace(country)
	if city == "" || country == "" {
		return errors.New("weather city and country are required")
	}
	u.WeatherCity, u.WeatherCountry = city, country
	return nil
}

func configureRecoveryProvisioning(ctx context.Context, ui prompt.UI, opt options, s *state) error {
	if !s.user.RecoveryEnable {
		return nil
	}

	// Canonical fresh installation owns recovery partition #3 on the
	// selected target disk. It must never ask the user for a second GPT
	// disk or for GjallarOS release-signing private/public key files.
	//
	// Release signing and per-device recovery identity are separate trust
	// roles. Fresh storage provisioning only creates the canonical recovery
	// storage here.
	if !s.existing {
		s.recoveryDisk = ""
		s.recoveryPartition = ""
		s.recoverySigningKey = ""
		s.recoverySigningPublicKey = ""
		return nil
	}
	if opt.recoveryDisk != "" && opt.recoveryPartition != "" {
		return errors.New("choose either --recovery-disk or --recovery-partition, not both")
	}
	explicitTarget := opt.recoveryDisk != "" || opt.recoveryPartition != ""
	s.recoveryDisk = opt.recoveryDisk
	s.recoveryPartition = opt.recoveryPartition
	s.recoverySigningKey = opt.recoverySigningKey
	s.recoverySigningPublicKey = opt.recoverySigningPublicKey
	if s.recoveryPartition == "" {
		s.recoveryPartition = findRecoveryPartition(ctx)
	}

	// Existing encrypted-Btrfs installations may provision recovery storage,
	// but the running root is never resized in place. GJAL-67 discovery and
	// planning are read-only on the live system; any required contraction is
	// executed only from the trusted one-shot maintenance context with the
	// target root mounted at /mnt.
	//
	// An already-existing dedicated recovery partition remains usable without
	// entering the resize path.

	// recoveryPartitionEnable authorizes creation/resizing of dedicated
	// recovery storage. An already-existing canonical recovery partition
	// remains usable even when automatic partition provisioning is disabled.
	if s.recoveryPartition == "" &&
		!s.user.RecoveryPartitionEnable &&
		!explicitTarget {
		if s.user.EndpointManagedDevice && !s.existing {
			return errors.New(
				"managed fresh installation requires creation of an independent recovery partition",
			)
		}
		return nil
	}

	if explicitTarget {
		if !filepath.IsAbs(s.recoverySigningKey) {
			return errors.New("--recovery-signing-key must be an absolute runtime path")
		}
		if !filepath.IsAbs(s.recoverySigningPublicKey) {
			return errors.New("--recovery-signing-public-key must be an absolute runtime path")
		}
		if !s.user.SecureBootEnable {
			return errors.New("physical recovery partition provisioning requires Secure Boot")
		}
		return nil
	}
	if s.recoveryPartition != "" {
		yes, err := ui.Confirm(ctx, "Install/update the recovery environment on "+s.recoveryPartition+" after a successful rebuild? That partition will be formatted.", false)
		if err != nil {
			return err
		}
		if !yes {
			s.recoveryPartition = ""
			if s.user.EndpointManagedDevice && !s.existing {
				return errors.New("managed fresh installation requires an independent recovery partition")
			}
			return nil
		}
	} else {
		// Fresh canonical installation owns recovery partition #3 on the
		// validated target disk. Do not ask for a second recovery disk here.
		//
		// Existing-layout GJAL-65 resize orchestration is a separate path and
		// still rediscovers the actual Btrfs/LUKS/GPT topology before mutation.
		s.recoveryDisk = ""
	}
	if s.recoverySigningKey == "" {
		var err error
		s.recoverySigningKey, err = ui.Value(ctx, "Runtime path to the offline recovery-image Ed25519 private signing key", "")
		if err != nil {
			return err
		}
	}
	if s.recoverySigningPublicKey == "" {
		var err error
		s.recoverySigningPublicKey, err = ui.Value(ctx, "Runtime path to the separately pinned recovery-image Ed25519 public key", "")
		if err != nil {
			return err
		}
	}
	if !filepath.IsAbs(s.recoverySigningKey) {
		return errors.New("recovery signing key path must be absolute")
	}
	if !filepath.IsAbs(s.recoverySigningPublicKey) {
		return errors.New("recovery signing public-key path must be absolute")
	}
	if !s.user.SecureBootEnable {
		return errors.New("physical recovery partition provisioning requires Secure Boot")
	}
	return nil
}

func discoverInstalledRecoveryTopology(
	ctx context.Context,
	installedRoot string,
) (recoveryresize.Topology, error) {
	mounted, err := exec.CommandContext(
		ctx,
		"findmnt",
		"-nro",
		"SOURCE",
		"--target",
		installedRoot,
	).Output()
	if err != nil {
		return recoveryresize.Topology{}, fmt.Errorf(
			"discover installed root mapping: %w",
			err,
		)
	}

	mapping := strings.TrimSpace(string(mounted))
	if !strings.HasPrefix(mapping, "/dev/mapper/") {
		return recoveryresize.Topology{}, fmt.Errorf(
			"recovery partitioning currently requires an encrypted Btrfs root; mounted root source is %q",
			mapping,
		)
	}

	mappingName := filepath.Base(mapping)

	if err := attached(ctx, "sudo", "-v"); err != nil {
		return recoveryresize.Topology{}, fmt.Errorf("authorize encrypted-root inspection: %w", err)
	}

	status, err := exec.CommandContext(
		ctx,
		"sudo",
		"-n",
		"cryptsetup",
		"status",
		mappingName,
	).Output()
	if err != nil {
		return recoveryresize.Topology{}, fmt.Errorf(
			"inspect installed LUKS mapping %s: %w",
			mappingName,
			err,
		)
	}

	rootPartition := ""
	for _, line := range strings.Split(string(status), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) == 2 && fields[0] == "device:" {
			rootPartition = fields[1]
			break
		}
	}

	if rootPartition == "" {
		return recoveryresize.Topology{}, fmt.Errorf(
			"cryptsetup status for %s did not report a backing device",
			mappingName,
		)
	}

	resolvedRoot, err := filepath.EvalSymlinks(rootPartition)
	if err != nil {
		return recoveryresize.Topology{}, fmt.Errorf(
			"resolve installed root partition %s: %w",
			rootPartition,
			err,
		)
	}

	parentRaw, err := exec.CommandContext(
		ctx,
		"lsblk",
		"-dnro",
		"PKNAME",
		resolvedRoot,
	).Output()
	if err != nil {
		return recoveryresize.Topology{}, fmt.Errorf(
			"resolve installed root parent disk: %w",
			err,
		)
	}

	parent := strings.TrimSpace(string(parentRaw))
	if parent == "" {
		return recoveryresize.Topology{}, fmt.Errorf(
			"installed root partition %s has no parent disk",
			resolvedRoot,
		)
	}
	if !strings.HasPrefix(parent, "/dev/") {
		parent = filepath.Join("/dev", parent)
	}

	topology, err := recoveryresize.DiscoverTopology(
		ctx,
		recoveryresize.SystemRunner(),
		recoveryresize.DiscoveryInput{
			RootMountpoint:            installedRoot,
			ExpectedDiskPath:          parent,
			ExpectedRootPartitionPath: resolvedRoot,
			ExpectedMappingPath:       mapping,
		},
	)
	if err != nil {
		return recoveryresize.Topology{}, err
	}

	return topology, nil
}

func prepareInstalledRecoveryStorage(
	ctx context.Context,
	ui prompt.UI,
	out io.Writer,
	installedRoot string,
	s *state,
) (bool, error) {
	topology, err := discoverInstalledRecoveryTopology(ctx, installedRoot)
	if err != nil {
		return false, err
	}

	plan, err := recoveryprovision.BuildExistingPlan(
		ctx,
		recoveryresize.SystemRunner(),
		topology,
	)
	if err != nil {
		return false, fmt.Errorf(
			"build existing-layout recovery plan: %w",
			err,
		)
	}

	result, err := recoveryprovision.Prepare(
		ctx,
		recoveryresize.SystemRunner(),
		recoveryprovision.Input{
			Plan:                      plan,
			UI:                        ui,
			Out:                       out,
			RootMountpoint:            "/",
			ExpectedRootPartitionPath: topology.RootPartitionPath,
			ExpectedMappingPath:       topology.MappingPath,
			Unattended:                s.user.UnattendedInstall,
		},
	)
	if err != nil {
		return false, err
	}

	switch result.Status {
	case recoveryprovision.StatusReady:
		s.recoveryPartition = result.RecoveryPartition
		fmt.Fprintf(
			out,
			"PASS: existing encrypted-Btrfs layout has exact 12 GiB JODS-RECOVERY at %s\n",
			result.RecoveryPartition,
		)
		return false, nil

	case recoveryprovision.StatusResizeRequired:
		if !s.user.UnattendedInstall {
			confirmed, err := ui.Confirm(
				ctx,
				"Recovery storage requires shrinking the encrypted Btrfs root. Reboot once into trusted maintenance mode and continue safely?",
				false,
			)
			if err != nil {
				return false, err
			}
			if !confirmed {
				return false, errors.New(
					"recovery-storage maintenance was not authorized; no disk changes were made",
				)
			}
		}

		fmt.Fprintln(
			out,
			"PASS: live root inspection complete; no mutation was performed",
		)
		fmt.Fprintln(
			out,
			"PLAN: install and boot the one-shot /mnt maintenance environment",
		)

		return true, nil

	default:
		return false, fmt.Errorf(
			"unexpected installed recovery provisioning state %q",
			result.Status,
		)
	}
}

func findRecoveryPartition(ctx context.Context) string {
	out, err := exec.CommandContext(ctx, "lsblk", "-pnro", "PATH,PARTLABEL,LABEL").Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && (fields[1] == "JODS-RECOVERY" || fields[1] == "JODSRECOV") {
			return fields[0]
		}
		if len(fields) >= 3 && fields[2] == "JODSRECOV" {
			return fields[0]
		}
	}
	return ""
}

func provisionRecoveryPartition(ctx context.Context, root string, s state) error {
	if s.recoveryDisk == "" && s.recoveryPartition == "" {
		return nil
	}
	createScript := filepath.Join(root, "scripts", "recovery", "create-partition.sh")
	installScript := filepath.Join(root, "scripts", "recovery", "install-partition.sh")
	signScript := filepath.Join(root, "scripts", "recovery", "sign-image.sh")
	if s.recoveryDisk != "" {
		if err := attached(ctx, "sudo", createScript, s.recoveryDisk); err != nil {
			return fmt.Errorf("create recovery partition: %w", err)
		}
		s.recoveryPartition = findRecoveryPartition(ctx)
		if s.recoveryPartition == "" {
			return errors.New("recovery partition creation completed but JODS-RECOVERY was not detected")
		}
	}
	cmd := exec.CommandContext(ctx, "nix", "build", root+"#gjallar-recovery-iso", "--no-link", "--print-out-paths")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("build recovery image: %w", err)
	}
	storePath := strings.TrimSpace(string(output))
	images, err := filepath.Glob(filepath.Join(storePath, "iso", "*.iso"))
	if err != nil || len(images) != 1 {
		return fmt.Errorf("recovery build produced %d ISO images", len(images))
	}
	releaseDir, err := os.MkdirTemp("", "gjallar-recovery-release-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(releaseDir)
	if err := attached(ctx, signScript, images[0], s.recoverySigningKey, s.recoverySigningPublicKey, releaseDir); err != nil {
		return fmt.Errorf("sign recovery image release: %w", err)
	}
	manifest := filepath.Join(releaseDir, filepath.Base(images[0])+".manifest")
	if err := attached(ctx, "sudo", installScript, s.recoveryPartition, images[0], manifest, manifest+".sig", filepath.Join(releaseDir, "recovery-signing-public.pem")); err != nil {
		return fmt.Errorf("install recovery partition: %w", err)
	}
	return nil
}

func detectAndRenderState(
	ctx context.Context,
	root string,
	s *state,
	hardware discovery.Hardware,
	resolvedDevice oddc.Resolved,
) error {
	u := s.user
	if u.AIAgentMode == "" {
		u.AIAgentMode = "workspace"
	}
	for _, item := range []struct {
		role  string
		value *string
	}{{"normal", &u.BackgroundNormal}} {
		resolved, err := background.Resolve(ctx, root, u.DotfilesDir, item.role, *item.value)
		if err != nil {
			return err
		}
		*item.value = resolved
	}
	g, err := graphics.Detect(ctx)
	if err != nil {
		return err
	}
	ai := profile.Result{Model: "qwen3-coder:30b", ContextTokens: 8192}
	if u.AIEnable {
		ai, err = profile.Detect(ctx, func() string {
			if s.preset {
				return filepath.Join(root, "user.config.json")
			}
			return ""
		}())
		if err != nil {
			return err
		}
	}
	system, err := detectedNixSystem()
	if err != nil {
		return err
	}

	rootPasswordFile := s.render.RootPasswordFile
	s.render = nixrender.FromUser(u)
	s.render.System = system
	s.render.TouchscreenEnable = s.touchscreen
	s.render.PenTabletEnable = s.penTablet
	s.render.OrientationSensorEnable = s.orientationSensor
	s.render.RootPasswordFile = rootPasswordFile
	s.render.ODDCModel = resolvedDevice.ModelID
	s.render.GraphicsBusID = g.BusID
	s.render.GraphicsIntegratedBusID = g.IntegratedBusID
	s.render.AIModel = ai.Model
	s.render.AIAccelerationProfile = ai.AccelerationProfile
	s.render.AIContextTokens = ai.ContextTokens
	s.render.AIVRAMMB = ai.VRAMMB
	s.render.WMs = []string{"hyprland"}

	return nil
}

func configureSecrets(context.Context, string, *state, io.Writer) error {
	return nil
}

func controlBinary() string {
	exe, err := os.Executable()
	if err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "gjallarctl")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return "gjallarctl"
}
func controlOutput(ctx context.Context, binary string, stderr io.Writer, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Stdin = os.Stdin
	cmd.Stderr = stderr
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("gjallarctl %s: %w", strings.Join(args, " "), err)
	}
	return string(out), nil
}
func controlAttached(ctx context.Context, binary string, args ...string) error {
	return attached(ctx, binary, args...)
}
func attached(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
func privilegedFileExists(ctx context.Context, path string) bool {
	return exec.CommandContext(ctx, "sudo", "test", "-s", path).Run() == nil
}
func secureBootNeedsFirmwareReboot(next secureboot.Continuation) bool {
	return next == secureboot.ContinuationEnroll || next == secureboot.ContinuationEnable
}
func requireNixOS(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if line == "ID=nixos" || line == `ID="nixos"` {
			return nil
		}
	}
	return errors.New("installer must run on NixOS")
}
func existingInstall(root string) bool {
	data, err := os.ReadFile(filepath.Join(root, "generated", "state.nix"))
	return err == nil && (strings.Contains(string(data), "Generated by gjallarctl") || strings.Contains(string(data), "Generated by scripts/installation/install.sh"))
}
func first(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
func validateSelections(u config.User, o discovery.Options) error {
	contains := func(values []string, want string) bool {
		for _, value := range values {
			if value == want {
				return true
			}
		}
		return false
	}
	for _, check := range []struct {
		name, value string
		allowed     []string
	}{{"shell", u.Shell, o.Shells}} {
		if !contains(check.allowed, check.value) {
			return fmt.Errorf("unsupported %s: %q", check.name, check.value)
		}
	}
	for _, value := range u.Editors {
		if !contains(o.Editors, value) {
			return fmt.Errorf("unsupported editor: %q", value)
		}
	}
	for _, value := range u.Browsers {
		if !contains(o.Browsers, value) {
			return fmt.Errorf("unsupported browser: %q", value)
		}
	}
	if !filepath.IsAbs(u.DotfilesDir) {
		return fmt.Errorf("dotfiles path must be absolute: %q", u.DotfilesDir)
	}
	return nil
}
func fail(out io.Writer, err error) int { fmt.Fprintln(out, "ERROR:", err); return 1 }
