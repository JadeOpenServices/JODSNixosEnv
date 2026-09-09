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
	"github.com/bakanura/gjallarOS/internal/hardware/network"
	"github.com/bakanura/gjallarOS/internal/input/xkb"
	"github.com/bakanura/gjallarOS/internal/installer/background"
	"github.com/bakanura/gjallarOS/internal/installer/bootstrap"
	"github.com/bakanura/gjallarOS/internal/installer/config"
	"github.com/bakanura/gjallarOS/internal/installer/deploy"
	"github.com/bakanura/gjallarOS/internal/installer/discovery"
	"github.com/bakanura/gjallarOS/internal/installer/firmware"
	"github.com/bakanura/gjallarOS/internal/installer/geolocation"
	"github.com/bakanura/gjallarOS/internal/installer/hardwareconfig"
	"github.com/bakanura/gjallarOS/internal/installer/localgit"
	"github.com/bakanura/gjallarOS/internal/installer/nixrender"
	"github.com/bakanura/gjallarOS/internal/installer/prompt"
	"github.com/bakanura/gjallarOS/internal/installer/recoveryprovision"
	"github.com/bakanura/gjallarOS/internal/installer/recoveryresize"
	"github.com/bakanura/gjallarOS/internal/installer/release"
	"github.com/bakanura/gjallarOS/internal/installer/secureboot"
	"github.com/bakanura/gjallarOS/internal/installercheck"
)

type options struct {
	recovery                                                 bool
	repo                                                     string
	skipHardware, refreshHardware, noRebuild, acceptExisting bool
	targetDisk                                               string
	recoveryDisk, recoveryPartition, recoverySigningKey      string
	recoverySigningPublicKey                                 string
}
type state struct {
	user                     config.User
	render                   nixrender.Settings
	passthroughIDs           []string
	preset, existing         bool
	control                  string
	touchscreen              bool
	penTablet                bool
	recoveryDisk             string
	recoveryPartition        string
	recoverySigningKey       string
	recoverySigningPublicKey string
}

const installerSecureBootResumeMarker = "/var/lib/gjallarOS/installer-resume-after-secure-boot"

var detectNetworkLocation = geolocation.Detect
var runJODSCommand = attached

func Run(ctx context.Context, args []string, in io.Reader, out, errOut io.Writer) int {
	f := flag.NewFlagSet("gjallar-installer", flag.ContinueOnError)
	f.SetOutput(errOut)
	cwd, _ := os.Getwd()
	opt := options{}
	f.StringVar(&opt.repo, "repo", cwd, "GjallarOS repository")
	f.BoolVar(&opt.skipHardware, "skip-hardware", false, "skip hardware generation")
	f.BoolVar(&opt.refreshHardware, "refresh-hardware", false, "regenerate hardware configuration")
	f.BoolVar(&opt.noRebuild, "no-rebuild", false, "do not install a boot generation")
	f.BoolVar(&opt.acceptExisting, "accept-existing", false, "allow an existing GjallarOS installation to be updated")
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
	presetPath := filepath.Join(root, "user.config.json")
	if _, err := os.Stat(presetPath); err == nil {
		s.user, err = config.Load(presetPath)
		if err != nil {
			return fail(errOut, err)
		}
		s.preset = true
	}
	if opt.recovery && !opt.acceptExisting {
		// The embedded repository is installation source material, not evidence
		// that the recovery environment itself is an installed GjallarOS system.
		// Recovery without --accept-existing is the explicit fresh operation.
		s.existing = false
	} else {
		s.existing = existingInstall(root)
	}
	if s.existing && !opt.acceptExisting {
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
	} else if code := prepareHost(ctx, ui, opt, s, out, errOut); code != 0 {
		return code
	}
	hardware := discovery.DetectHardware("/sys")
	s.touchscreen = hardware.Touchscreen
	s.penTablet = hardware.PenTablet
	fmt.Fprintf(out, "Touchscreen detected: %t\n", hardware.Touchscreen)
	fmt.Fprintf(out, "Pen/tablet detected: %t\n", hardware.PenTablet)
	choices, err := discovery.Discover(root, s.preset, hardware)
	if err != nil {
		return fail(errOut, err)
	}
	if s.preset && s.user.Profile == "auto" {
		s.user.Profile = machineProfileForHardware(hardware)
		if s.user.Profile == "" {
			if s.user.UnattendedInstall {
				return fail(errOut, errors.New("machine profile could not be detected; set profile to desktop or laptop in user.config.json"))
			}
			s.user.Profile, err = ui.Choice(ctx, "Machine profile could not be detected", "desktop", []string{"desktop", "laptop"})
			if err != nil {
				return fail(errOut, err)
			}
		} else {
			fmt.Fprintf(out, "Machine profile detected: %s\n", s.user.Profile)
		}
	}
	if s.preset {
		normalizePreset(&s.user, root)
	} else if err := collectInteractive(ctx, ui, root, hardware, choices, &s.user); err != nil {
		return fail(errOut, err)
	}
	if err := configureWeatherLocation(ctx, ui, &s.user, out); err != nil {
		return fail(errOut, err)
	}
	if s.user.FrameworkEnable && s.user.SecureBootPrompt && !s.user.EndpointManagedDevice {
		s.user.SecureBootEnable, err = ui.Confirm(ctx, "Prepare Framework Secure Boot and recovery keys?", false)
		if err != nil {
			return fail(errOut, err)
		}
	}
	// Resolve the managed-device security contract before recovery
	// provisioning so a managed preset cannot silently skip the physical
	// recovery path by leaving its individual booleans false.
	normalizeManagementSafety(&s.user)

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
	if err := detectAndRenderState(ctx, root, &s); err != nil {
		return fail(errOut, err)
	}
	if !s.preset && s.user.NemuEnable && len(s.passthroughIDs) > 0 {
		enabled, err := ui.Confirm(ctx, "Pass the detected dedicated GPU through to Nemu? This removes it from the host.", false)
		if err != nil {
			return fail(errOut, err)
		}
		s.render.NemuGPUPassthrough = enabled
		if enabled {
			s.render.NemuGPUIDs = append([]string(nil), s.passthroughIDs...)
		}
	}
	if err := configureSecrets(ctx, root, &s, errOut); err != nil {
		return fail(errOut, err)
	}
	fmt.Fprintf(out, "\nSelected: profile=%s hostname=%s user=%s shell=%s theme=%s\n", s.user.Profile, s.user.Hostname, s.user.Username, s.user.Shell, s.user.Theme)
	write := s.user.WriteConfig
	if !s.preset {
		write, err = ui.Confirm(ctx, "Write configuration to "+filepath.Join(root, "settings.nix")+"?", false)
		if err != nil {
			return fail(errOut, err)
		}
	}
	if !write {
		fmt.Fprintln(out, "Nothing changed.")
		return 0
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
		path, err := controlOutput(ctx, s.control, errOut, "installer", "work-password", "--username", "root", "--apply")
		if err != nil {
			return fail(errOut, err)
		}
		lines := strings.Fields(strings.TrimSpace(path))
		if len(lines) > 0 {
			s.render.RootPasswordFile = lines[len(lines)-1]
		}
		fmt.Fprint(out, path)
	}
	if s.render.WorkUserEnable {
		if s.render.EndpointManagedDevice {
			if s.user.WorkUserPasswordFile == "" {
				return fail(errOut, errors.New("managed work account requires workUserPasswordFile provisioned by JODS"))
			}
			s.render.WorkUserPasswordFile = s.user.WorkUserPasswordFile
		} else if workPasswordPath := filepath.Join("/var/lib/gjallarOS/passwords", s.render.WorkUsername+".hash"); s.existing && privilegedFileExists(ctx, workPasswordPath) {
			s.render.WorkUserPasswordFile = workPasswordPath
			fmt.Fprintln(out, "Existing work-account password hash retained.")
		} else {
			path, err := controlOutput(ctx, s.control, errOut, "installer", "work-password", "--username", s.render.WorkUsername, "--apply")
			if err != nil {
				return fail(errOut, err)
			}
			lines := strings.Fields(strings.TrimSpace(path))
			if len(lines) > 0 {
				s.render.WorkUserPasswordFile = lines[len(lines)-1]
			}
			fmt.Fprint(out, path)
		}
	}
	settingsPath := filepath.Join(root, "settings.nix")
	if err := nixrender.WriteAtomic(settingsPath, s.render); err != nil {
		return fail(errOut, err)
	}
	fmt.Fprintln(out, "Wrote", settingsPath)
	hardwarePath := filepath.Join(root, "profiles", s.user.Profile, "hardware-configuration.nix")
	skip := opt.skipHardware || (s.existing && !opt.refreshHardware)
	if !skip {
		fmt.Fprintln(out, "PLAN: generate and atomically replace", hardwarePath)
		if backup, err := hardwareconfig.Generate(ctx, root, hardwarePath, time.Now()); err != nil {
			return fail(errOut, err)
		} else if backup != "" {
			fmt.Fprintln(out, "Backup:", backup)
		}
		if s.render.EndpointManagedDevice {
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
	runRebuild := s.user.RunRebuild && !opt.noRebuild
	if !s.preset && !opt.noRebuild {
		runRebuild, err = ui.Confirm(ctx, "Install the next NixOS boot generation now?", false)
		if err != nil {
			return fail(errOut, err)
		}
	}
	if runRebuild {
		if s.existing {
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
					&s,
				)
				if err != nil {
					return fail(errOut, err)
				}

				if maintenanceRequired {
					s.recoveryPartition = ""
				}
			}

			target, err := deploy.Target(root, s.user.Hostname)
			if err != nil {
				return fail(errOut, err)
			}
			fmt.Fprintln(out, "PLAN: validate then install", target)
			if err := deploy.Apply(ctx, target); err != nil {
				return fail(errOut, err)
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

				if err := attached(
					ctx,
					"sudo",
					"gjallar-recovery-maintenance-next",
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
				return fail(errOut, err)
			}

			if s.recoveryDisk != "" || s.recoveryPartition != "" {
				return fail(
					errOut,
					errors.New(
						"canonical fresh installation owns recovery storage on --target-disk; "+
							"do not combine it with --recovery-disk or --recovery-partition",
					),
				)
			}

			fresh, err := runFreshBareMetal(
				ctx,
				ui,
				root,
				targetDisk,
				s.user.Hostname,
				s.user.RecoveryEnable &&
					s.user.RecoveryPartitionEnable,
				[]string{
					s.render.RootPasswordFile,
					s.render.WorkUserPasswordFile,
				},
				out,
			)
			if err != nil {
				return fail(errOut, err)
			}

			if fresh.RecoveryPartition != "" {
				s.recoveryPartition = fresh.RecoveryPartition
				fmt.Fprintf(
					out,
					"Canonical recovery storage prepared at %s; recovery identity/release provisioning is handled by the appropriate local or JODS trust flow.\n",
					fresh.RecoveryPartition,
				)
			}
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

		next, err := secureboot.VerifyAndArmEnrollment(ctx)
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

			if err := ui.SecureBootEnableHandoff(ctx); err != nil {
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

			if err := ui.SecureBootFirmwareHandoff(ctx); err != nil {
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

func prepareHost(ctx context.Context, ui prompt.UI, opt options, s state, out, errOut io.Writer) int {
	expected, actual, err := release.Inspect(opt.repo, "/etc/os-release")
	if err != nil {
		return fail(errOut, err)
	}
	if expected != actual {
		fmt.Fprintf(out, "NixOS %s detected; target is %s.\n", actual, expected)
		fmt.Fprintf(out, "ACTION: automatically aligning the base system to pinned NixOS %s.\n", expected)

		if err := release.Align(ctx, expected); err != nil {
			return fail(errOut, err)
		}

		_, active, inspectErr := release.Inspect(opt.repo, "/run/current-system/etc/os-release")
		if inspectErr != nil {
			return fail(errOut, inspectErr)
		}
		if active != expected {
			return fail(errOut, fmt.Errorf("release alignment completed, but NixOS %s is not active; detected %s", expected, active))
		}

		fmt.Fprintf(out, "NixOS release aligned: %s\n", active)
	}
	// A completed GjallarOS installation is flake-owned. Rebuilding the
	// bootstrap /etc/nixos/configuration.nix on a rerun would switch the live
	// machine back to its pre-install base generation before deploying the
	// requested flake generation.
	if s.existing {
		fmt.Fprintln(out, "Existing GjallarOS installation detected; skipping /etc/nixos bootstrap rebuild.")
	} else {
		configPath := "/etc/nixos/configuration.nix"
		data, err := os.ReadFile(configPath)
		if err != nil {
			return fail(errOut, err)
		}
		plan := bootstrap.Build(data, s.preset)
		if len(plan.Missing) > 0 {
			fmt.Fprintf(out, "Missing helpers/options: %s\nProposed configuration:\n%s\n", strings.Join(plan.Missing, ", "), plan.Updated)
			if !s.user.EndpointManagedDevice {
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
	if !s.preset {
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

func machineProfileForHardware(hardware discovery.Hardware) string {
	switch hardware.FormFactor {
	case "desktop":
		return "desktop"
	case "laptop":
		return "laptop"
	default:
		return ""
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
	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x86_64"
	} else if arch == "arm64" {
		arch = "aarch64"
	}
	var err error
	if u.System, err = ui.Choice(ctx, "System architecture", arch+"-linux", []string{"x86_64-linux", "aarch64-linux"}); err != nil {
		return err
	}
	profileDefault := machineProfileForHardware(hardware)
	if profileDefault == "" || !containsValue(o.Profiles, profileDefault) {
		profileDefault = first(o.Profiles)
	}
	if u.Profile, err = ui.Choice(ctx, "Profile", profileDefault, o.Profiles); err != nil {
		return err
	}
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
		endpointDefault := "https://admin.oss-ad.eu:1666"
		if localDevelopment {
			endpointDefault = "https://192.0.2.193:1666"
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
	u.WorkUserEnable, err = ui.Confirm(ctx, "Create a separate work account?", false)
	if err != nil {
		return err
	}
	if u.WorkUserEnable {
		u.WorkUsername = u.Username + "-corp"
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
	u.USBGuardEnable, err = ui.Confirm(ctx, "Enable USBGuard? New devices will be blocked until permitted.", false)
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
	u.Theme, err = ui.Choice(ctx, "Theme", first(o.Themes), o.Themes)
	if err != nil {
		return err
	}
	u.DockerEnable, err = ui.Confirm(ctx, "Enable Docker daemon? Docker access is root-equivalent.", false)
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
	if hardware.LaptopVendor == "framework" || strings.HasPrefix(u.Profile, "framework") {
		u.FrameworkEnable = true
		u.FrameworkModel, err = ui.Choice(ctx, "Framework model", "13", []string{"13", "16", "12"})
		if err != nil {
			return err
		}
	}
	u.WriteConfig = true
	u.RunRebuild = true
	return nil
}

func collectProjectTools(ctx context.Context, ui prompt.UI, u *config.User) error {
	var err error

	u.PlaneEnable, err = ui.Confirm(ctx, "Enable Plane integration?", false)
	if err != nil {
		return err
	}
	if u.PlaneEnable {
		u.PlaneHost, err = ui.Value(ctx, "Plane host", "")
		if err != nil {
			return err
		}
	}

	u.DrawioEnable, err = ui.Confirm(ctx, "Enable Draw.io integration?", false)
	if err != nil {
		return err
	}
	if u.DrawioEnable {
		u.DrawioSelfHosted, err = ui.Confirm(ctx, "Use a self-hosted Draw.io server?", false)
		if err != nil {
			return err
		}

		if u.DrawioSelfHosted {
			u.DrawioHost, err = ui.Value(ctx, "Draw.io endpoint", "")
			if err != nil {
				return err
			}
		}
	}

	return config.NormalizeProjectTools(u)
}

func normalizePreset(u *config.User, root string) {
	if u.DotfilesDir == "" {
		u.DotfilesDir = filepath.Join("/home", u.Username, "Documents", "gjallarOS")
	}
	u.DotfilesDir = strings.ReplaceAll(u.DotfilesDir, "usernamehere", u.Username)
	if u.WorkUserEnable {
		u.WorkUsername = u.Username + "-corp"
	}
	if normalized, err := xkb.Normalize(u.KeyboardLayout); err == nil {
		u.KeyboardLayout, u.KeyboardVariant = normalized.Name, normalized.Variant
	}
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
) (recoveryresize.Topology, error) {
	mounted, err := exec.CommandContext(
		ctx,
		"findmnt",
		"-nro",
		"SOURCE",
		"--target",
		"/",
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

	status, err := exec.CommandContext(
		ctx,
		"sudo",
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
			RootMountpoint:            "/",
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
	s *state,
) (bool, error) {
	topology, err := discoverInstalledRecoveryTopology(ctx)
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

func detectAndRenderState(ctx context.Context, root string, s *state) error {
	u := s.user
	if u.AIAgentMode == "" {
		u.AIAgentMode = "workspace"
	}
	for _, item := range []struct {
		role  string
		value *string
	}{{"normal", &u.BackgroundNormal}, {"work", &u.BackgroundWork}, {"gaming", &u.BackgroundGaming}} {
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
	wifi, err := network.DetectWiFiDriver(ctx)
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
	s.passthroughIDs = append([]string(nil), g.PassthroughIDs...)
	passthrough := u.NemuGPUPassthrough && len(g.PassthroughIDs) > 0
	s.render = nixrender.Settings{System: u.System, Profile: u.Profile, Hostname: u.Hostname, Username: u.Username, Timezone: u.Timezone, Locale: u.Locale, KeyboardLayout: u.KeyboardLayout, KeyboardVariant: u.KeyboardVariant, WeatherCity: u.WeatherCity, WeatherCountry: u.WeatherCountry, TouchpadWorkspaceSwipe: u.TouchpadWorkspaceSwipe, TouchscreenEnable: s.touchscreen, PenTabletEnable: s.penTablet, ClamshellEnable: u.ClamshellEnable, USBGuardEnable: u.USBGuardEnable, Name: u.Name, Email: u.Email, GitHubUsername: u.GitHubUsername, DotfilesDir: u.DotfilesDir, WorkUserEnable: u.WorkUserEnable, WorkUsername: u.WorkUsername, DockerEnable: u.DockerEnable, DebugFunctions: u.DebugFunctions, Shell: u.Shell, Editors: u.Editors, Browsers: u.Browsers, PreferredEditor: u.PreferredEditor, PreferredBrowser: u.PreferredBrowser, PlaneEnable: u.PlaneEnable, PlaneHost: u.PlaneHost, DrawioEnable: u.DrawioEnable, DrawioSelfHosted: u.DrawioSelfHosted, DrawioHost: u.DrawioHost, BackgroundNormal: u.BackgroundNormal, BackgroundWork: u.BackgroundWork, BackgroundGaming: u.BackgroundGaming, EnableScrobbling: u.EnableScrobbling, EnableLastfm: u.EnableLastfm, EnableListenbrainz: u.EnableListenbrainz, LastfmUsername: u.LastfmUsername, ListenbrainzUsername: u.ListenbrainzUsername, FrameworkEnable: u.FrameworkEnable, FrameworkModel: u.FrameworkModel, GraphicsVendor: g.Vendor, GraphicsType: g.Type, GraphicsCompute: g.Compute, GraphicsBusID: g.BusID, GraphicsIntegratedBusID: g.IntegratedBusID, WiFiDriver: wifi, AIEnable: u.AIEnable, AIModel: ai.Model, AIAccelerationProfile: ai.AccelerationProfile, AIAgentMode: u.AIAgentMode, AIContextTokens: ai.ContextTokens, AIVRAMMB: ai.VRAMMB, NemuEnable: u.NemuEnable, NemuGPUPassthrough: passthrough, LUKSTPM2Enable: u.LUKSTPM2Enable, RecoveryEnable: u.RecoveryEnable, RecoveryPartitionEnable: u.RecoveryPartitionEnable, JODSPrebootLockEnable: u.JODSPrebootLockEnable, SecureBootEnable: u.SecureBootEnable, EndpointManagedDevice: u.EndpointManagedDevice, JODSEndpoint: u.JODSEndpoint, JODSPolicySigningPublicKey: u.JODSPolicySigningKey, JODSRecoveryCommandSigningPublicKey: u.JODSRecoverySigningKey, JODSEnrollmentMode: u.JODSEnrollmentMode, JODSAllowInsecureTLS: u.JODSAllowInsecureTLS, JODSDeviceClass: u.JODSDeviceClass, JODSDesktopProfile: u.JODSDesktopProfile, JODSFingerprintEnrollmentAllowed: u.JODSFingerprintEnroll, WMs: []string{"hyprland"}, Theme: u.Theme}
	if passthrough {
		s.render.NemuGPUIDs = g.PassthroughIDs
	}
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
	data, err := os.ReadFile(filepath.Join(root, "settings.nix"))
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
	}{{"profile", u.Profile, o.Profiles}, {"shell", u.Shell, o.Shells}, {"theme", u.Theme, o.Themes}} {
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
