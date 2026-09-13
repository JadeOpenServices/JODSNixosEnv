// gjallarctl is the safe, non-interactive control tool for GjallarOS.
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/bakanura/gjallarOS/internal/hardware/deviceprobe"
	"github.com/bakanura/gjallarOS/internal/hardware/deviceprobe/devices/hp/zbookx2g4"
	"github.com/bakanura/gjallarOS/internal/installer/repojson"
	"io"
	"math/rand/v2"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	osuser "os/user"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"
	"unicode/utf8"
	"unsafe"

	"github.com/bakanura/gjallarOS/internal/ai/agentexec"
	"github.com/bakanura/gjallarOS/internal/ai/modelbroker"
	aipolicy "github.com/bakanura/gjallarOS/internal/ai/policy"
	"github.com/bakanura/gjallarOS/internal/ai/profile"
	"github.com/bakanura/gjallarOS/internal/ai/research"
	"github.com/bakanura/gjallarOS/internal/hardware/graphics"
	"github.com/bakanura/gjallarOS/internal/hardware/network"
	"github.com/bakanura/gjallarOS/internal/input/xkb"
	"github.com/bakanura/gjallarOS/internal/installer/background"
	"github.com/bakanura/gjallarOS/internal/installer/bootstrap"
	"github.com/bakanura/gjallarOS/internal/installer/config"
	"github.com/bakanura/gjallarOS/internal/installer/deploy"
	"github.com/bakanura/gjallarOS/internal/installer/diskcrypto"
	"github.com/bakanura/gjallarOS/internal/installer/firmware"
	"github.com/bakanura/gjallarOS/internal/installer/hardwareconfig"
	"github.com/bakanura/gjallarOS/internal/installer/localgit"
	"github.com/bakanura/gjallarOS/internal/installer/nixrender"
	"github.com/bakanura/gjallarOS/internal/installer/oddcvalidation"
	"github.com/bakanura/gjallarOS/internal/installer/policy"
	"github.com/bakanura/gjallarOS/internal/installer/release"
	"github.com/bakanura/gjallarOS/internal/installer/secrets"
	"github.com/bakanura/gjallarOS/internal/installer/secureboot"
	"github.com/bakanura/gjallarOS/internal/installer/workpassword"
	"github.com/bakanura/gjallarOS/internal/installercheck"
	"github.com/bakanura/gjallarOS/internal/preset"
)

const (
	rebuildUIEnter = "\x1b[?1049h\x1b[?25l"
	rebuildUILeave = "\x1b[?25h\x1b[?1049l"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		printUsage(stdout)
		return 0
	}

	switch args[0] {
	case "installer":
		return runInstaller(args[1:], stdout, stderr)
	case "check":
		return runCheck(args[1:], stdout, stderr)
	case "oddc":
		return runODDC(args[1:], stdout, stderr)
	case "detect":
		return runDetect(args[1:], stdout, stderr)
	case "device-probe":
		return runDeviceProbe(args[1:], stdout, stderr)
	case "hyprland-rotate":
		return runHyprlandRotate(args[1:], stdout, stderr)
	case "ai":
		return runAI(args[1:], stdout, stderr)
	case "normalize":
		return runNormalize(args[1:], stdout, stderr)
	case "preset":
		return runPreset(args[1:], stdout, stderr)
	case "rebuild":
		return runRebuild(args[1:], stdout, stderr)
	case "update":
		return runUpdate(args[1:], stdout, stderr)
	case "cleanup":
		return runCleanup(args[1:], stdout, stderr)
	case "cleanup-old-generations":
		return runCleanupOld(args[1:], stdout, stderr)
	case "thermal-status":
		return runThermalStatus(args[1:], stdout, stderr)
	case "thermal-test":
		return runThermalTest(args[1:], stdout, stderr)
	case "helpme":
		return runHelpme(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "ERROR: unknown command %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func runInstaller(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "Usage: gjallarctl installer {check-secrets|deploy|firmware|generate-hardware|policy|protect-local|release|render|resolve-background|work-password}")
		return 2
	}
	if args[0] == "render" {
		return runRender(args[1:], stdout, stderr)
	}
	if args[0] == "protect-local" {
		return runProtectLocal(args[1:], stdout, stderr)
	}
	if args[0] == "generate-hardware" {
		return runGenerateHardware(args[1:], stdout, stderr)
	}
	if args[0] == "deploy" {
		return runDeploy(args[1:], stdout, stderr)
	}
	if args[0] == "resolve-background" {
		return runResolveBackground(args[1:], stdout, stderr)
	}
	if args[0] == "release" {
		return runRelease(args[1:], stdout, stderr)
	}
	if args[0] == "check-secrets" {
		return runCheckSecrets(args[1:], stdout, stderr)
	}
	if args[0] == "configure-scrobbling" {
		return runConfigureScrobbling(args[1:], stdout, stderr)
	}
	if args[0] == "firmware" {
		return runFirmware(args[1:], stdout, stderr)
	}
	if args[0] == "secure-boot-enroll" {
		return runSecureBootEnroll(args[1:], stdout, stderr)
	}
	if args[0] == "secure-boot-verify-ownership" {
		return runSecureBootVerifyOwnership(args[1:], stdout, stderr)
	}
	if args[0] == "secure-boot-firmware-instructions" {
		return runSecureBootFirmwareInstructions(args[1:], stdout, stderr)
	}
	if args[0] == "tpm2-metadata-token-id" {
		return runTPM2MetadataTokenID(args[1:], stdout, stderr)
	}
	if args[0] == "tpm2-write-keyslot-record" {
		return runTPM2WriteKeyslotRecord(args[1:], stdout, stderr)
	}
	if args[0] == "work-password" {
		return runWorkPassword(args[1:], stdout, stderr)
	}
	if args[0] == "tpm2" {
		return runTPM2(args[1:], stdout, stderr)
	}
	if args[0] == "luks" {
		return runLUKS(args[1:], stdout, stderr)
	}
	if args[0] == "bootstrap" {
		return runBootstrap(args[1:], stdout, stderr)
	}
	if args[0] != "policy" {
		fmt.Fprintln(stderr, "Usage: gjallarctl installer {check-secrets|deploy|firmware|generate-hardware|policy|protect-local|release|render|resolve-background|work-password}")
		return 2
	}
	flags := flag.NewFlagSet("gjallarctl installer policy", flag.ContinueOnError)
	flags.SetOutput(stderr)
	configPath := flags.String("config", "user.config.json", "user configuration path")
	if err := flags.Parse(args[1:]); err != nil || flags.NArg() != 0 {
		return 2
	}
	user, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}
	features := policy.FromUser(user)
	fmt.Fprintf(stdout, "ai_enable=%t\nauto_reboot=%t\ndebug_functions=%t\ndocker_enable=%t\nclamshell_enable=%t\nusbguard_enable=%t\nnemu_enable=%t\nnemu_gpu_passthrough=%t\ntouchpad_workspace_swipe=%t\nwork_user_enable=%t\n",
		features.AIEnable, features.AutoReboot, features.DebugFunctions, features.DockerEnable,
		features.ClamshellEnable, features.USBGuardEnable, features.NemuEnable,
		features.NemuGPUPassthrough, features.TouchpadWorkspaceSwipe, features.WorkUserEnable)
	return 0
}

func runConfigureScrobbling(args []string, stdout, stderr io.Writer) int {
	f := flag.NewFlagSet("gjallarctl installer configure-scrobbling", flag.ContinueOnError)
	f.SetOutput(stderr)
	repo := f.String("repo", "", "repository path")
	username := f.String("username", "", "Linux username")
	lastfmEnabled := f.Bool("lastfm", false, "configure Last.fm")
	listenEnabled := f.Bool("listenbrainz", false, "configure ListenBrainz")
	lastfmUser := f.String("lastfm-username", "", "Last.fm username")
	listenUser := f.String("listenbrainz-username", "", "ListenBrainz username")
	if err := f.Parse(args); err != nil || f.NArg() != 0 {
		return 2
	}
	if !*lastfmEnabled && !*listenEnabled {
		fmt.Fprintln(stdout, "enable_scrobbling=false")
		return 0
	}
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: scrobbling credentials require a terminal: %v\n", err)
		return 1
	}
	defer tty.Close()
	reader := bufio.NewReader(tty)
	readUser := func(label, current string) (string, error) {
		if current != "" {
			return current, nil
		}
		fmt.Fprintf(tty, "%s username: ", label)
		value, err := reader.ReadString('\n')
		return strings.TrimSpace(value), err
	}
	var lastfmToken, listenToken string
	if *lastfmEnabled {
		*lastfmUser, err = readUser("Last.fm", *lastfmUser)
		if err == nil {
			lastfmToken, err = workpassword.ReadSecret(tty, reader, tty, "Last.fm token/password: ")
		}
		if err != nil {
			fmt.Fprintf(stderr, "ERROR: %v\n", err)
			return 1
		}
	}
	if *listenEnabled {
		*listenUser, err = readUser("ListenBrainz", *listenUser)
		if err == nil {
			listenToken, err = workpassword.ReadSecret(tty, reader, tty, "ListenBrainz token: ")
		}
		if err != nil {
			fmt.Fprintf(stderr, "ERROR: %v\n", err)
			return 1
		}
	}
	if strings.ContainsAny(*lastfmUser+*listenUser, "\r\n=") {
		fmt.Fprintln(stderr, "ERROR: invalid scrobbling username")
		return 2
	}
	key, recipient, err := secrets.EnsureAgeKey(context.Background(), *username)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}
	fmt.Fprintf(tty, "Using age key %s. Back it up securely.\n", key)
	if err := secrets.EncryptScrobbling(context.Background(), *repo, recipient, lastfmToken, listenToken); err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}
	lastfmToken, listenToken = "", ""
	fmt.Fprintf(stdout, "enable_scrobbling=true\nenable_lastfm=%t\nenable_listenbrainz=%t\nlastfm_username=%s\nlistenbrainz_username=%s\n", *lastfmEnabled, *listenEnabled, *lastfmUser, *listenUser)
	return 0
}

func runBootstrap(args []string, stdout, stderr io.Writer) int {
	f := flag.NewFlagSet("gjallarctl installer bootstrap", flag.ContinueOnError)
	f.SetOutput(stderr)
	configPath := f.String("config", "/etc/nixos/configuration.nix", "NixOS configuration")
	presetMode := f.Bool("preset", false, "omit graphical prompt dependency")
	apply := f.Bool("apply", false, "apply and rebuild")
	if err := f.Parse(args); err != nil || f.NArg() != 0 {
		return 2
	}
	data, err := os.ReadFile(*configPath)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}
	plan := bootstrap.Build(data, *presetMode)
	if len(plan.Missing) == 0 {
		fmt.Fprintln(stdout, "Bootstrap prerequisites already present.")
		if err := firmware.EnsureService(context.Background()); err != nil {
			fmt.Fprintf(stderr, "WARN: %v\n", err)
		}
		return 0
	}
	fmt.Fprintf(stdout, "Missing helpers/options: %s\nPackages: %s\n--- proposed %s ---\n%s\n--- end proposed ---\n", strings.Join(plan.Missing, ", "), strings.Join(plan.Packages, " "), *configPath, plan.Updated)
	if !*apply {
		return 3
	}
	backup, err := bootstrap.Apply(context.Background(), *configPath, plan.Updated, time.Now())
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		if backup != "" {
			fmt.Fprintf(stderr, "Backup remains at %s\n", backup)
		}
		return 1
	}
	if err := firmware.EnsureService(context.Background()); err != nil {
		fmt.Fprintf(stderr, "WARN: %v\n", err)
	}
	fmt.Fprintln(stdout, "Bootstrap rebuild succeeded; previous configuration backup removed.")
	return 0
}

func ttyConfirm(tty *os.File, reader *bufio.Reader, prompt string) (bool, bool, error) {
	fmt.Fprintf(tty, "%s [y/N/q] ", prompt)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false, false, err
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, false, nil
	case "q", "quit":
		return false, true, nil
	default:
		return false, false, nil
	}
}

func runTPM2(args []string, stdout, stderr io.Writer) int {
	f := flag.NewFlagSet("gjallarctl installer tpm2", flag.ContinueOnError)
	f.SetOutput(stderr)
	repo := f.String("repo", "", "repository path")
	hardware := f.String("hardware", "", "hardware configuration")
	if err := f.Parse(args); err != nil || f.NArg() != 0 {
		return 2
	}
	resolved, err := hardwareconfig.ValidateTarget(*repo, *hardware)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 2
	}
	mappingInfo, err := diskcrypto.MappingDetails(resolved)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}
	if mappingInfo.Name == "" {
		fmt.Fprintln(stdout, "luks_tpm2_enable=false\n[NOTE] No LUKS mapping found for TPM2 enrollment.")
		return 0
	}
	if !diskcrypto.TPMAvailable() {
		fmt.Fprintln(stdout, "luks_tpm2_enable=false\n[NOTE] No TPM2 device detected; keeping passphrase unlock.")
		return 0
	}
	device, err := diskcrypto.DetectDevice(context.Background())
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: refusing TPM2 enrollment: %v\n", err)
		return 1
	}
	if device == "" {
		fmt.Fprintln(stdout, "luks_tpm2_enable=false\n[NOTE] No active crypto_LUKS device found; skipping TPM2 enrollment.")
		return 0
	}
	if err := diskcrypto.VerifyDeviceIdentity(mappingInfo.Device, device); err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: TPM2 enrollment requires a terminal: %v\n", err)
		return 1
	}
	defer tty.Close()
	reader := bufio.NewReader(tty)
	yes, _, err := ttyConfirm(tty, reader, "Prepare TPM2 automatic unlock after GjallarOS Secure Boot and measured boot are verified? The passphrase remains as recovery.")
	if err != nil || !yes {
		fmt.Fprintln(stdout, "luks_tpm2_enable=false")
		return 0
	}
	if err := diskcrypto.EnableTPMConfig(resolved, mappingInfo.Name); err != nil {
		fmt.Fprintf(stderr, "ERROR: TPM2 preparation failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "PLAN: after Secure Boot verification, request the human LUKS passphrase and enroll this TPM against PCRLock on %s\n", device)
	fmt.Fprintln(stdout, "luks_tpm2_enable=true")
	return 0
}

func confirmSaved(tty *os.File, reader *bufio.Reader, label, key string) (bool, error) {
	for {
		fmt.Fprintf(tty, "\n%s\n%s\nIMPORTANT: save this key securely; it is never persisted.\n", label, key)
		yes, quit, err := ttyConfirm(tty, reader, "Have you saved it?")
		if err != nil {
			return false, err
		}
		if quit {
			return false, nil
		}
		if yes {
			again, quit, err := ttyConfirm(tty, reader, "Are you absolutely sure it is saved?")
			if err != nil {
				return false, err
			}
			if quit {
				return false, nil
			}
			if again {
				return true, nil
			}
		}
		fmt.Fprintln(tty, "Confirmation required; displaying the key again.")
	}
}

func runLUKS(args []string, stdout, stderr io.Writer) int {
	f := flag.NewFlagSet("gjallarctl installer luks", flag.ContinueOnError)
	f.SetOutput(stderr)
	repo := f.String("repo", "", "repository path")
	hardware := f.String("hardware", "", "hardware configuration")
	if err := f.Parse(args); err != nil || f.NArg() != 0 {
		return 2
	}
	resolved, err := hardwareconfig.ValidateTarget(*repo, *hardware)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 2
	}
	mappingInfo, err := diskcrypto.MappingDetails(resolved)
	if err != nil || mappingInfo.Name == "" {
		fmt.Fprintln(stdout, "[NOTE] No LUKS mapping exists; skipping LUKS configuration.")
		return 0
	}
	device, err := diskcrypto.DetectDevice(context.Background())
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: refusing LUKS configuration: %v\n", err)
		return 1
	}
	if device == "" {
		fmt.Fprintln(stdout, "[NOTE] No LUKS-encrypted device detected; skipping LUKS configuration.")
		return 0
	}
	if err := diskcrypto.VerifyDeviceIdentity(mappingInfo.Device, device); err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: LUKS configuration requires a terminal: %v\n", err)
		return 1
	}
	defer tty.Close()
	reader := bufio.NewReader(tty)
	yes, _, err := ttyConfirm(tty, reader, "Configure LUKS keys for "+device+"?")
	if err != nil || !yes {
		fmt.Fprintln(stdout, "LUKS configuration skipped; existing unlock methods remain unchanged.")
		return 0
	}
	old, err := workpassword.ReadSecret(tty, reader, tty, "Current LUKS key: ")
	if err != nil || old == "" {
		fmt.Fprintln(stdout, "No key supplied; existing configuration unchanged.")
		return 0
	}
	if err := diskcrypto.Verify(context.Background(), device, old); err != nil {
		old = ""
		fmt.Fprintln(stderr, "ERROR: supplied LUKS key rejected; no disk changes made.")
		return 1
	}
	yes, _, err = ttyConfirm(tty, reader, "Generate a new LUKS key and separate recovery key?")
	if err != nil || !yes {
		old = ""
		fmt.Fprintln(stdout, "Keeping the existing LUKS key.")
		return 0
	}
	newKey, err := diskcrypto.GenerateKey()
	if err != nil {
		return 1
	}
	saved, err := confirmSaved(tty, reader, "NEW LUKS KEY", newKey)
	if err != nil || !saved {
		old = ""
		newKey = ""
		fmt.Fprintln(stdout, "LUKS rotation cancelled; existing key remains unchanged.")
		return 0
	}
	recovery, err := diskcrypto.GenerateKey()
	if err != nil {
		return 1
	}
	saved, err = confirmSaved(tty, reader, "LUKS RECOVERY KEY", recovery)
	if err != nil || !saved {
		old = ""
		newKey = ""
		recovery = ""
		fmt.Fprintln(stdout, "LUKS rotation cancelled; existing key remains unchanged.")
		return 0
	}
	fmt.Fprintln(stdout, "PLAN: add and verify new key; add and verify recovery key; preserve every existing keyslot")
	if err := diskcrypto.AddKeysPreservingExisting(context.Background(), device, old, newKey, recovery); err != nil {
		old = ""
		newKey = ""
		recovery = ""
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}
	old = ""
	newKey = ""
	recovery = ""
	fmt.Fprintln(stdout, "LUKS enrollment complete: new and recovery keys verified; all previous keyslots preserved.")
	return 0
}

func runWorkPassword(args []string, stdout, stderr io.Writer) int {
	f := flag.NewFlagSet("gjallarctl installer work-password", flag.ContinueOnError)
	f.SetOutput(stderr)
	username := f.String("username", "", "work-account username")
	apply := f.Bool("apply", false, "prompt and store a missing password hash")
	if err := f.Parse(args); err != nil || f.NArg() != 0 {
		return 2
	}
	target, err := workpassword.Path(*username)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "PLAN: reuse %s when present; otherwise prompt on /dev/tty, hash, and install mode 0600\n", target)
	if !*apply {
		return 0
	}
	if workpassword.Exists(context.Background(), target) {
		fmt.Fprintf(stdout, "Reusing stored password hash for %s.\n", *username)
		fmt.Fprintln(stdout, target)
		return 0
	}
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: work-account password requires a terminal: %v\n", err)
		return 1
	}
	defer tty.Close()
	fmt.Fprintf(tty, "Set a password for %s (terminal only).\n", *username)
	password, err := workpassword.ReadConfirmedPassword(tty, tty, *username)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}
	hash, err := workpassword.Hash(context.Background(), password)
	password = ""
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}
	if err := workpassword.Store(context.Background(), target, hash); err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}
	hash = ""
	fmt.Fprintf(stdout, "Stored password hash for %s.\n%s\n", *username, target)
	return 0
}

func runFirmware(args []string, stdout, stderr io.Writer) int {
	f := flag.NewFlagSet("gjallarctl installer firmware", flag.ContinueOnError)
	f.SetOutput(stderr)
	apply := f.Bool("apply", false, "refresh metadata and apply updates")
	ensure := f.Bool("ensure-service", false, "start fwupd service")
	if err := f.Parse(args); err != nil || f.NArg() != 0 || (*apply && *ensure) {
		return 2
	}
	if !firmware.Available() {
		fmt.Fprintln(stdout, "[NOTE] fwupdmgr is unavailable; skipping firmware operations.")
		return 0
	}
	if *ensure {
		if err := firmware.EnsureService(context.Background()); err != nil {
			fmt.Fprintf(stderr, "WARN: %v\n", err)
		}
		return 0
	}
	fmt.Fprintln(stdout, "PLAN: sudo fwupdmgr refresh --force\nPLAN: sudo fwupdmgr update")
	if !*apply {
		return 0
	}
	result, err := firmware.Update(context.Background())
	if err != nil {
		fmt.Fprintf(stderr, "WARN: %v\n", err)
		return 0
	}
	if result == "none" {
		fmt.Fprintln(stdout, "No firmware updates available; continuing.")
	} else {
		fmt.Fprintln(stdout, "Firmware update process completed. A reboot may be required.")
	}
	return 0
}

func runSecureBootEnroll(args []string, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "Usage: gjallarctl installer secure-boot-enroll")
		return 2
	}

	result, err := secureboot.EnrollFirmware(context.Background())
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: Secure Boot firmware enrollment refused: %v\n", err)
		return 1
	}

	switch result {
	case secureboot.EnrollmentWaitingForSetupMode:
		fmt.Fprintln(
			stdout,
			"Secure Boot enrollment remains armed; firmware has not entered the required Setup Mode.",
		)
		return 0

	case secureboot.EnrollmentCompleted:
		fmt.Fprintln(
			stdout,
			"Secure Boot firmware enrollment completed from the trusted ODDC policy snapshot.",
		)
		return 0

	default:
		fmt.Fprintf(
			stderr,
			"ERROR: unexpected Secure Boot enrollment result %q\n",
			result,
		)
		return 1
	}
}

func runTPM2MetadataTokenID(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		fmt.Fprintln(stderr, "Usage: gjallarctl installer tpm2-metadata-token-id <metadata-json>")
		return 2
	}

	id, err := diskcrypto.ReadTPM2TokenID(args[0])
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: TPM2 metadata validation failed: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, id)
	return 0
}

func runTPM2WriteKeyslotRecord(args []string, stdout, stderr io.Writer) int {
	if len(args) != 4 {
		fmt.Fprintln(
			stderr,
			"Usage: gjallarctl installer tpm2-write-keyslot-record <metadata-json> <target-json> <device> <policy>",
		)
		return 2
	}

	if err := diskcrypto.WriteTPM2KeyslotRecord(
		args[0],
		args[1],
		args[2],
		args[3],
	); err != nil {
		fmt.Fprintf(stderr, "ERROR: TPM2 keyslot record failed: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, "TPM2 keyslot record written")
	return 0
}

func runSecureBootFirmwareInstructions(args []string, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(
			stderr,
			"Usage: gjallarctl installer secure-boot-firmware-instructions",
		)
		return 2
	}

	instructions, err := secureboot.FirmwareInstructions(
		secureboot.FirmwarePolicyPath,
	)
	if err != nil {
		fmt.Fprintf(
			stderr,
			"ERROR: Secure Boot firmware instructions unavailable: %v\n",
			err,
		)
		return 1
	}

	for _, instruction := range instructions {
		fmt.Fprintln(stdout, instruction)
	}

	return 0
}

func runSecureBootVerifyOwnership(args []string, stdout, stderr io.Writer) int {
	if len(args) > 1 {
		fmt.Fprintln(
			stderr,
			"Usage: gjallarctl installer secure-boot-verify-ownership [expected-stage]",
		)
		return 2
	}

	expectedStage := ""
	if len(args) == 1 {
		expectedStage = args[0]
	}

	if err := secureboot.VerifyOwnership(
		context.Background(),
		expectedStage,
	); err != nil {
		fmt.Fprintf(
			stderr,
			"ERROR: Secure Boot ownership verification failed: %v\n",
			err,
		)
		return 1
	}

	fmt.Fprintln(
		stdout,
		"PASS: active GjallarOS PK/KEK/db ownership verified.",
	)
	return 0
}

func runCheckSecrets(args []string, stdout, stderr io.Writer) int {
	f := flag.NewFlagSet("gjallarctl installer check-secrets", flag.ContinueOnError)
	f.SetOutput(stderr)
	repo := f.String("repo", "", "repository path")
	username := f.String("username", "", "Linux username")
	if err := f.Parse(args); err != nil || f.NArg() != 0 {
		return 2
	}
	status, err := secrets.Check(context.Background(), *repo, *username)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}
	if !status.SecretsExist {
		fmt.Fprintln(stdout, "SOPS: no secrets/default.yaml found; continuing without secrets.")
		return 0
	}
	if status.KeyExists {
		fmt.Fprintf(stdout, "SOPS: age key found at %s\n", status.KeyPath)
	} else {
		fmt.Fprintf(stdout, "SOPS: age key missing at %s\nCreate it before rebuilding with age-keygen.\n", status.KeyPath)
	}
	if status.Decrypts {
		fmt.Fprintln(stdout, "SOPS: secrets/default.yaml decrypts successfully.")
	} else if status.SOPSAvailable && status.KeyExists {
		fmt.Fprintln(stdout, "SOPS: secrets/default.yaml cannot be decrypted with this user's key.\nRe-encrypt it with your age recipient before applying the configuration.")
	} else {
		fmt.Fprintln(stdout, "SOPS: install sops and age, then verify secrets/default.yaml before rebuilding.")
	}
	return 0
}

func runRelease(args []string, stdout, stderr io.Writer) int {
	f := flag.NewFlagSet("gjallarctl installer release", flag.ContinueOnError)
	f.SetOutput(stderr)
	repo := f.String("repo", "", "repository path")
	osRelease := f.String("os-release", "/etc/os-release", "OS release file")
	apply := f.Bool("apply", false, "align a mismatched release")
	if err := f.Parse(args); err != nil || f.NArg() != 0 {
		return 2
	}
	expected, actual, err := release.Inspect(*repo, *osRelease)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}
	if actual == expected {
		fmt.Fprintf(stdout, "NixOS release verified: %s\n", actual)
		return 0
	}
	fmt.Fprintf(stdout, "NixOS %s detected; this checkout targets NixOS %s.\n", actual, expected)
	fmt.Fprintf(stdout, "PLAN: sudo nix-channel --add https://channels.nixos.org/nixos-%s nixos\nPLAN: sudo nix-channel --update nixos\nPLAN: sudo nixos-rebuild boot --upgrade\n", expected)
	if !*apply {
		return 3
	}

	const nixosConfig = "/etc/nixos/configuration.nix"
	if _, err := os.Stat(nixosConfig); err != nil {
		fmt.Fprintf(
			stderr,
			"ERROR: NixOS configuration unavailable at %s: %v\n",
			nixosConfig,
			err,
		)
		return 1
	}

	if err := release.Align(
		context.Background(),
		expected,
		nixosConfig,
	); err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}

	fmt.Fprintf(
		stdout,
		"NixOS %s staged for the next boot. Reboot to activate it.\n",
		expected,
	)
	return 0
}

func runResolveBackground(args []string, stdout, stderr io.Writer) int {
	f := flag.NewFlagSet("gjallarctl installer resolve-background", flag.ContinueOnError)
	f.SetOutput(stderr)
	repo := f.String("repo", "", "repository path")
	dotfiles := f.String("dotfiles", "", "configured dotfiles path")
	role := f.String("role", "", "normal, work, or gaming")
	value := f.String("value", "", "path or HTTPS URL")
	if err := f.Parse(args); err != nil || f.NArg() != 0 {
		return 2
	}
	resolved, err := background.Resolve(context.Background(), *repo, *dotfiles, *role, *value)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, resolved)
	return 0
}

func runDeploy(args []string, stdout, stderr io.Writer) int {
	f := flag.NewFlagSet("gjallarctl installer deploy", flag.ContinueOnError)
	f.SetOutput(stderr)
	repo := f.String("repo", "", "absolute repository path")
	hostname := f.String("hostname", "", "flake configuration name")
	apply := f.Bool("apply", false, "install the boot generation")
	if err := f.Parse(args); err != nil || f.NArg() != 0 {
		return 2
	}
	target, err := deploy.Target(*repo, *hostname)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "PLAN: sudo nixos-rebuild dry-build --flake %s --show-trace\nPLAN: sudo nixos-rebuild boot --flake %s\n", target, target)
	if !*apply {
		return 0
	}
	if err := deploy.Apply(context.Background(), target); err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "NixOS boot generation installed.")
	return 0
}

func runGenerateHardware(args []string, stdout, stderr io.Writer) int {
	f := flag.NewFlagSet("gjallarctl installer generate-hardware", flag.ContinueOnError)
	f.SetOutput(stderr)
	repo := f.String("repo", "", "absolute repository path")
	target := f.String("target", "", "absolute hardware configuration path")
	apply := f.Bool("apply", false, "generate and replace the target")
	if err := f.Parse(args); err != nil || f.NArg() != 0 {
		return 2
	}
	resolved, err := hardwareconfig.ValidateTarget(*repo, *target)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "PLAN: run nixos-generate-config --show-hardware-config\nPLAN: atomically replace %s\n", resolved)
	if !*apply {
		return 0
	}
	backup, err := hardwareconfig.Generate(context.Background(), *repo, *target, time.Now())
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}
	if backup != "" {
		fmt.Fprintf(stdout, "Backup: %s\n", backup)
	}
	fmt.Fprintf(stdout, "Generated hardware configuration: %s\n", resolved)
	return 0
}

func runProtectLocal(args []string, stdout, stderr io.Writer) int {
	f := flag.NewFlagSet("gjallarctl installer protect-local", flag.ContinueOnError)
	f.SetOutput(stderr)
	repo := f.String("repo", "", "absolute repository path")
	hardware := f.String("hardware", "", "optional hardware configuration path")
	if err := f.Parse(args); err != nil || f.NArg() != 0 {
		return 2
	}
	if !filepath.IsAbs(*repo) {
		fmt.Fprintln(stderr, "ERROR: --repo must be an absolute path")
		return 2
	}
	protected, err := localgit.Protect(context.Background(), *repo, *hardware)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}
	for _, path := range protected {
		fmt.Fprintf(stdout, "Protected local config: %s\n", path)
	}
	return 0
}

type stringList []string

func (s *stringList) String() string     { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error { *s = append(*s, v); return nil }

func runRender(args []string, stdout, stderr io.Writer) int {
	f := flag.NewFlagSet("gjallarctl installer render", flag.ContinueOnError)
	f.SetOutput(stderr)
	var s nixrender.Settings
	output := f.String("output", "settings.nix", "output path")
	f.StringVar(&s.System, "system", "", "system")
	f.StringVar(&s.Profile, "profile", "", "profile")
	f.StringVar(&s.Hostname, "hostname", "", "hostname")
	f.StringVar(&s.Username, "username", "", "username")
	f.StringVar(&s.Timezone, "timezone", "", "timezone")
	f.StringVar(&s.Locale, "locale", "", "locale")
	f.StringVar(&s.KeyboardLayout, "keyboard-layout", "", "keyboard layout")
	f.StringVar(&s.KeyboardVariant, "keyboard-variant", "", "keyboard variant")
	f.BoolVar(&s.TouchpadWorkspaceSwipe, "touchpad-workspace-swipe", false, "")
	f.BoolVar(&s.TouchscreenEnable, "touchscreen-enable", false, "")
	f.BoolVar(&s.PenTabletEnable, "pen-tablet-enable", false, "")
	f.BoolVar(&s.ClamshellEnable, "clamshell-enable", false, "")
	f.BoolVar(&s.USBGuardEnable, "usbguard-enable", false, "")
	f.StringVar(&s.Name, "name", "", "name")
	f.StringVar(&s.Email, "email", "", "email")
	f.StringVar(&s.GitHubUsername, "github-username", "", "GitHub username")
	f.StringVar(&s.DotfilesDir, "dotfiles-dir", "", "dotfiles directory")
	f.BoolVar(&s.WorkUserEnable, "work-user-enable", false, "")
	f.StringVar(&s.WorkUsername, "work-username", "", "")
	f.StringVar(&s.WorkUserPasswordFile, "work-user-password-file", "", "")
	f.BoolVar(&s.DockerEnable, "docker-enable", false, "")
	f.BoolVar(&s.DebugFunctions, "debug-functions", false, "")
	f.StringVar(&s.Shell, "shell", "", "")
	f.Var((*stringList)(&s.Editors), "editor", "repeatable editor")
	f.Var((*stringList)(&s.Browsers), "browser", "repeatable browser")
	f.StringVar(&s.PreferredEditor, "preferred-editor", "", "")
	f.StringVar(&s.PreferredBrowser, "preferred-browser", "", "")
	f.BoolVar(&s.PlaneEnable, "plane-enable", false, "")
	f.StringVar(&s.PlaneHost, "plane-host", "", "")
	f.BoolVar(&s.DrawioEnable, "drawio-enable", false, "")
	f.BoolVar(&s.DrawioSelfHosted, "drawio-self-hosted", false, "")
	f.StringVar(&s.DrawioHost, "drawio-host", "", "")
	f.StringVar(&s.BackgroundNormal, "background-normal", "", "")
	f.StringVar(&s.BackgroundWork, "background-work", "", "")
	f.StringVar(&s.BackgroundGaming, "background-gaming", "", "")
	f.BoolVar(&s.EnableScrobbling, "enable-scrobbling", false, "")
	f.BoolVar(&s.EnableLastfm, "enable-lastfm", false, "")
	f.BoolVar(&s.EnableListenbrainz, "enable-listenbrainz", false, "")
	f.StringVar(&s.LastfmUsername, "lastfm-username", "", "")
	f.StringVar(&s.ListenbrainzUsername, "listenbrainz-username", "", "")
	f.BoolVar(&s.FrameworkEnable, "framework-enable", false, "")
	f.StringVar(&s.FrameworkModel, "framework-model", "", "")
	f.StringVar(&s.DeviceProfile, "device-profile", "", "")
	f.Var((*stringList)(&s.DeviceLayers), "device-layer", "repeatable resolved oddc device layer")
	f.StringVar(&s.DeviceSysVendor, "device-sys-vendor", "", "")
	f.StringVar(&s.DeviceProductName, "device-product-name", "", "")
	f.StringVar(&s.DeviceProductVersion, "device-product-version", "", "")
	f.StringVar(&s.DeviceBoardVendor, "device-board-vendor", "", "")
	f.StringVar(&s.DeviceBoardName, "device-board-name", "", "")
	f.StringVar(&s.DeviceBoardVersion, "device-board-version", "", "")
	f.StringVar(&s.GraphicsVendor, "graphics-vendor", "", "")
	f.StringVar(&s.GraphicsType, "graphics-type", "", "")
	f.BoolVar(&s.GraphicsCompute, "graphics-compute", false, "")
	f.StringVar(&s.GraphicsBusID, "graphics-bus-id", "", "")
	f.StringVar(&s.GraphicsIntegratedBusID, "graphics-integrated-bus-id", "", "")
	f.StringVar(&s.WiFiDriver, "wifi-driver", "", "")
	f.BoolVar(&s.AIEnable, "ai-enable", false, "")
	f.StringVar(&s.AIModel, "ai-model", "", "")
	f.StringVar(&s.AIAgentMode, "ai-agent-mode", "workspace", "")
	f.IntVar(&s.AIContextTokens, "ai-context-tokens", 0, "")
	f.IntVar(&s.AIVRAMMB, "ai-vram-mb", 0, "")
	f.BoolVar(&s.NemuEnable, "nemu-enable", false, "")
	f.BoolVar(&s.NemuGPUPassthrough, "nemu-gpu-passthrough", false, "")
	f.Var((*stringList)(&s.NemuGPUIDs), "nemu-gpu-id", "repeatable GPU ID")
	f.BoolVar(&s.LUKSTPM2Enable, "luks-tpm2-enable", false, "")
	f.BoolVar(&s.RecoveryEnable, "recovery-enable", false, "")
	f.BoolVar(&s.JODSPrebootLockEnable, "jods-preboot-lock-enable", false, "")
	f.BoolVar(&s.SecureBootEnable, "secure-boot-enable", false, "")
	f.BoolVar(&s.EndpointManagedDevice, "endpoint-managed-device", false, "")
	f.StringVar(&s.JODSEndpoint, "jods-endpoint", "", "")
	f.StringVar(&s.JODSPolicySigningPublicKey, "jods-policy-signing-public-key", "", "")
	f.StringVar(&s.JODSRecoveryCommandSigningPublicKey, "jods-recovery-command-signing-public-key", "", "")
	f.StringVar(&s.JODSEnrollmentMode, "jods-enrollment-mode", "", "")
	f.BoolVar(&s.JODSAllowInsecureTLS, "jods-allow-insecure-tls", false, "")
	f.StringVar(&s.JODSDeviceClass, "jods-device-class", "", "")
	f.StringVar(&s.JODSDesktopProfile, "jods-desktop-profile", "", "")
	f.BoolVar(&s.JODSFingerprintEnrollmentAllowed, "jods-fingerprint-enrollment-allowed", false, "")
	f.Var((*stringList)(&s.WMs), "wm", "repeatable window manager")
	f.StringVar(&s.Theme, "theme", "", "")
	if err := f.Parse(args); err != nil || f.NArg() != 0 {
		return 2
	}
	if *output == "" || !filepath.IsAbs(*output) {
		fmt.Fprintln(stderr, "ERROR: --output must be an absolute path")
		return 2
	}
	if err := nixrender.WriteAtomic(*output, s); err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Wrote %s\n", *output)
	return 0
}

func runPreset(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || (args[0] != "validate" && args[0] != "get" && args[0] != "list" && args[0] != "bool") {
		fmt.Fprintln(stderr, "Usage: gjallarctl preset {validate|get|list|bool} --config PATH [--key NAME]")
		return 2
	}
	flags := flag.NewFlagSet("gjallarctl preset "+args[0], flag.ContinueOnError)
	flags.SetOutput(stderr)
	config := flags.String("config", "user.config.json", "preset JSON path")
	key := flags.String("key", "", "preset field name")
	if err := flags.Parse(args[1:]); err != nil || flags.NArg() != 0 || (args[0] != "validate" && *key == "") {
		return 2
	}
	document, err := preset.Load(*config)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}
	switch args[0] {
	case "validate":
		return 0
	case "get":
		value, err := document.String(*key)
		if err == nil {
			fmt.Fprintln(stdout, value)
		}
		return presetResult(err, stderr)
	case "list":
		values, err := document.Strings(*key)
		if err == nil {
			for _, value := range values {
				fmt.Fprintln(stdout, value)
			}
		}
		return presetResult(err, stderr)
	case "bool":
		value, err := document.Bool(*key)
		if err == nil {
			fmt.Fprintln(stdout, value)
		}
		return presetResult(err, stderr)
	default:
		return 2
	}
}

func presetResult(err error, stderr io.Writer) int {
	if err == nil {
		return 0
	}
	fmt.Fprintf(stderr, "ERROR: %v\n", err)
	return 1
}

func runNormalize(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "keyboard" {
		fmt.Fprintln(stderr, "Usage: gjallarctl normalize keyboard --layout VALUE")
		return 2
	}
	flags := flag.NewFlagSet("gjallarctl normalize keyboard", flag.ContinueOnError)
	flags.SetOutput(stderr)
	layout := flags.String("layout", "", "XKB keyboard layout")
	if err := flags.Parse(args[1:]); err != nil || flags.NArg() != 0 {
		return 2
	}
	result, err := xkb.Normalize(*layout)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "layout=%s\nvariant=%s\n", result.Name, result.Variant)
	return 0
}

func runAIModelServe(args []string, stdout, stderr io.Writer) int {
	f := flag.NewFlagSet(
		"gjallarctl ai model-serve",
		flag.ContinueOnError,
	)

	f.SetOutput(stderr)

	socket := f.String(
		"socket",
		"/run/gjallar-ai-model/model.sock",
		"Unix socket",
	)

	upstream := f.String(
		"upstream",
		"http://127.0.0.1:11434",
		"Ollama upstream",
	)

	model := f.String(
		"model",
		"gjallaros-caveman-ai",
		"only exposed model",
	)

	username := f.String(
		"user",
		"",
		"only authorized desktop user",
	)

	cgroupPrefix := f.String(
		"cgroup-prefix",
		"/system.slice/gjallar-ai-session@",
		"required system-service cgroup prefix",
	)

	if err := f.Parse(args); err != nil || f.NArg() != 0 {
		return 2
	}

	if *username == "" {
		fmt.Fprintln(
			stderr,
			"ERROR: --user is required",
		)
		return 2
	}

	account, err := osuser.Lookup(*username)
	if err != nil {
		fmt.Fprintf(
			stderr,
			"ERROR: resolve agent user: %v\n",
			err,
		)
		return 1
	}

	uid64, err := strconv.ParseUint(
		account.Uid,
		10,
		32,
	)

	if err != nil {
		fmt.Fprintf(
			stderr,
			"ERROR: invalid agent uid: %v\n",
			err,
		)
		return 1
	}

	err = modelbroker.Serve(
		context.Background(),
		modelbroker.Config{
			SocketPath:           *socket,
			Upstream:             *upstream,
			Model:                *model,
			AllowedUID:           uint32(uid64),
			RequiredCgroupPrefix: *cgroupPrefix,
			MaxRequestBytes:      16 << 20,
			Timeout:              15 * time.Minute,
		},
	)

	if err != nil {
		fmt.Fprintf(
			stderr,
			"ERROR: model broker: %v\n",
			err,
		)
		return 1
	}

	return 0
}

func runAI(args []string, stdout, stderr io.Writer) int {
	if len(args) > 0 && args[0] == "model-serve" {
		return runAIModelServe(args[1:], stdout, stderr)
	}

	if len(args) == 0 {
		fmt.Fprintln(stderr, "Usage: gjallarctl ai {profile|tool|approve|research-serve}")
		return 2
	}
	if args[0] == "tool" {
		return runAITool(args[1:], stdout, stderr)
	}
	if args[0] == "approve" {
		return runAIApprove(args[1:], stdout, stderr)
	}
	if args[0] == "research-serve" {
		return runAIResearchServe(args[1:], stdout, stderr)
	}
	if args[0] != "profile" {
		return 2
	}
	flags := flag.NewFlagSet("gjallarctl ai profile", flag.ContinueOnError)
	flags.SetOutput(stderr)
	config := flags.String("config", "user.config.json", "user configuration path")
	if err := flags.Parse(args[1:]); err != nil || flags.NArg() != 0 {
		return 2
	}
	result, err := profile.Detect(context.Background(), *config)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: AI profile detection failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "profile=%s\nmodel=%s\ncontext_tokens=%d\nvram_mb=%d\nram_gb=%d\ncpu_cores=%d\narchitecture=%s\ngpu_vendor=%s\ngpu_type=%s\n",
		result.Profile, result.Model, result.ContextTokens, result.VRAMMB, result.RAMGB, result.CPUCores, result.Architecture, result.GPUVendor, result.GPUType)
	return 0
}

func runAITool(args []string, stdout, stderr io.Writer) int {
	f := flag.NewFlagSet("gjallarctl ai tool", flag.ContinueOnError)
	f.SetOutput(stderr)
	workspace := f.String("workspace", ".", "active workspace")
	mode := f.String("mode", "workspace", "policy mode")
	approval := f.String("approval", "", "one-use approval file")
	audit := f.String("audit", "", "JSONL audit file")
	timeout := f.Duration("timeout", 10*time.Minute, "finite execution timeout")
	if err := f.Parse(args); err != nil || f.NArg() < 1 {
		return 2
	}
	if err := agentexec.Run(context.Background(), agentexec.Request{Workspace: *workspace, Tool: f.Arg(0), Args: f.Args()[1:], Mode: *mode, ApprovalPath: *approval, AuditPath: *audit, Timeout: *timeout}, stdout, stderr); err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}
	return 0
}

func runAIApprove(args []string, stdout, stderr io.Writer) int {
	f := flag.NewFlagSet("gjallarctl ai approve", flag.ContinueOnError)
	f.SetOutput(stderr)
	workspace := f.String("workspace", ".", "active workspace")
	grant := f.String("grant", "", "approval file")
	if err := f.Parse(args); err != nil || f.NArg() < 1 || *grant == "" {
		return 2
	}
	info, err := os.Stdin.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice == 0 {
		fmt.Fprintln(stderr, "ERROR: approval requires a trusted interactive terminal")
		return 1
	}
	fmt.Fprintf(stderr, "Approve exact action %q with arguments %q? [y/N] ", f.Arg(0), f.Args()[1:])
	line, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	if strings.ToLower(strings.TrimSpace(line)) != "y" {
		fmt.Fprintln(stderr, "Denied.")
		return 1
	}
	if err := aipolicy.WriteGrant(*grant, *workspace, f.Arg(0), f.Args()[1:], time.Now()); err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}
	fmt.Fprintln(stdout, "Exact one-use approval created; expires in five minutes.")
	return 0
}

func runAIResearchServe(args []string, stdout, stderr io.Writer) int {
	f := flag.NewFlagSet("gjallarctl ai research-serve", flag.ContinueOnError)
	f.SetOutput(stderr)
	socket := f.String("socket", "/run/gjallar-ai/research.sock", "Unix socket")
	maxBytes := f.Int64("max-bytes", 2<<20, "maximum response")
	if err := f.Parse(args); err != nil || f.NArg() != 0 {
		return 2
	}
	_ = os.Remove(*socket)
	if err := os.MkdirAll(filepath.Dir(*socket), 0750); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	listener, err := net.Listen("unix", *socket)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	defer listener.Close()
	_ = os.Chmod(*socket, 0666)
	b := research.Broker{AllowedHosts: []string{"nixos.org", "github.com", "docs.ollama.com", "opencode.ai"}, MaxBytes: *maxBytes, Timeout: 15 * time.Second}
	server := &http.Server{Handler: research.Handler(b), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 30 * time.Second}
	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func runDetect(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 || (args[0] != "graphics" && args[0] != "network") {
		fmt.Fprintln(stderr, "Usage: gjallarctl detect {graphics|network}")
		return 2
	}
	if args[0] == "network" {
		driver, err := network.DetectWiFiDriver(context.Background())
		if err != nil {
			fmt.Fprintf(stderr, "ERROR: network detection failed: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "wifi_driver=%s\n", driver)
		return 0
	}
	result, err := graphics.Detect(context.Background())
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: graphics detection failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "vendor=%s\ntype=%s\ncompute=%t\nbus=%s\nintegrated_bus=%s\npassthrough_ids=%s\n",
		result.Vendor, result.Type, result.Compute, result.BusID, result.IntegratedBusID,
		strings.Join(result.PassthroughIDs, " "))
	return 0
}

func runDeviceProbe(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "Usage: gjallarctl device-probe {refresh|diagnose}")
		return 2
	}

	if args[0] == "diagnose" {
		return runDeviceProbeDiagnose(args[1:], stdout, stderr)
	}

	if args[0] != "refresh" {
		fmt.Fprintln(stderr, "Usage: gjallarctl device-probe {refresh|diagnose}")
		return 2
	}

	flags := flag.NewFlagSet("gjallarctl device-probe refresh", flag.ContinueOnError)
	flags.SetOutput(stderr)

	output := flags.String(
		"output",
		"/run/gjallarOS/device-probe.json",
		"device probe snapshot path",
	)

	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "ERROR: device-probe refresh accepts no positional arguments")
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	snapshot, err := deviceprobe.Collect(ctx, "/sys")
	if err != nil {
		fmt.Fprintf(stderr, "FAIL: device probe collection: %v\n", err)
		return 1
	}

	if err := deviceprobe.WriteSnapshot(*output, snapshot); err != nil {
		fmt.Fprintf(stderr, "FAIL: device probe snapshot: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "PASS: device probe snapshot written to %s\n", *output)
	return 0
}

func runDeviceProbeDiagnose(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "zbook-x2-g4" {
		fmt.Fprintln(
			stderr,
			"Usage: gjallarctl device-probe diagnose zbook-x2-g4 [--input PATH]",
		)
		return 2
	}

	flags := flag.NewFlagSet(
		"gjallarctl device-probe diagnose zbook-x2-g4",
		flag.ContinueOnError,
	)
	flags.SetOutput(stderr)

	input := flags.String(
		"input",
		"/run/gjallarOS/device-probe.json",
		"device probe snapshot path",
	)
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "ERROR: unexpected positional arguments")
		return 2
	}

	snapshot, err := deviceprobe.ReadSnapshot(*input)
	if err != nil {
		fmt.Fprintf(stderr, "FAIL: snapshot: %v\n", err)
		return 1
	}

	report := zbookx2g4.Evaluate(snapshot)
	failed := false

	for _, result := range report.Results {
		fmt.Fprintf(
			stdout,
			"%s: %s: %s\n",
			result.Status,
			result.Gate,
			result.Detail,
		)
		if result.Status == zbookx2g4.StatusFail {
			failed = true
		}
	}

	if failed {
		return 1
	}

	return 0
}

func runODDC(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "ERROR: oddc requires a subcommand")
		return 2
	}

	switch args[0] {
	case "validate-device":
		return runODDCValidateDevice(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "ERROR: unknown oddc command %q\n", args[0])
		return 2
	}
}

func runODDCValidateDevice(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gjallarctl oddc validate-device", flag.ContinueOnError)
	flags.SetOutput(stderr)

	repo := flags.String("repo", ".", "GjallarOS repository root")

	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "ERROR: oddc validate-device accepts no positional arguments")
		return 2
	}

	root, err := installercheck.ResolveRepository(*repo)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 2
	}

	return runODDCValidateDeviceResolved(root, stdout, stderr)
}

type oddcValidationRunner func(
	context.Context,
	string,
	oddcvalidation.CommandRunner,
) (oddcvalidation.Report, oddcvalidation.DeviceContext, error)

func runODDCValidateDeviceResolved(
	root string,
	stdout io.Writer,
	stderr io.Writer,
) int {
	report, device, err := oddcvalidation.Run(
		context.Background(),
		root,
		nil,
	)
	if err == nil {
		pinnedRelease, releaseErr := release.Expected(root)
		if releaseErr != nil {
			err = releaseErr
		} else {
			validation, metadataErr := oddcvalidation.ValidationMetadata(
				report,
				device,
				pinnedRelease,
				time.Now(),
			)
			if metadataErr != nil {
				err = metadataErr
			} else {
				authority := oddcvalidation.CheckContributorAuthority(
					context.Background(),
					root,
					"bakanura/JODSNixosEnv",
				)
				mode, finalizeErr := oddcvalidation.FinalizeValidation(
					context.Background(),
					root,
					device.Resolved.Device.ID,
					validation,
					authority,
				)
				if finalizeErr != nil {
					err = finalizeErr
				} else {
					fmt.Fprintf(stdout, "PASS: validation recorded (%s)\n", mode)
				}
			}
		}
	}

	for _, result := range report.Results {
		status := "PASS"
		if !result.Passed {
			status = "FAIL"
		}
		if result.Details != "" {
			fmt.Fprintf(stdout, "%s: %s - %s\n", status, result.Gate, result.Details)
		} else {
			fmt.Fprintf(stdout, "%s: %s\n", status, result.Gate)
		}
	}

	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, "PASS: ODDC real-device validation complete")
	return 0
}

func runODDCValidateDeviceWith(
	root string,
	stdout io.Writer,
	stderr io.Writer,
	runValidation oddcValidationRunner,
) int {
	report, _, err := runValidation(
		context.Background(),
		root,
		nil,
	)

	for _, result := range report.Results {
		status := "PASS"
		if !result.Passed {
			status = "FAIL"
		}

		if result.Details != "" {
			fmt.Fprintf(
				stdout,
				"%s: %s - %s\n",
				status,
				result.Gate,
				result.Details,
			)
			continue
		}

		fmt.Fprintf(stdout, "%s: %s\n", status, result.Gate)
	}

	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}

	fmt.Fprintln(stdout, "PASS: ODDC real-device validation complete")
	return 0
}

func runCheck(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gjallarctl check", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repo := flags.String("repo", ".", "GjallarOS repository root")
	timeout := flags.Duration("timeout", 5*time.Second, "maximum time for each safe external check")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "ERROR: check accepts no positional arguments")
		return 2
	}
	if *timeout <= 0 || *timeout > 30*time.Second {
		fmt.Fprintln(stderr, "ERROR: --timeout must be greater than zero and no more than 30s")
		return 2
	}

	root, err := installercheck.ResolveRepository(*repo)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	report := installercheck.Check(ctx, root)
	for _, finding := range report.Findings {
		fmt.Fprintf(stdout, "%s: %s\n", finding.Level, finding.Message)
	}
	if report.Failed() {
		fmt.Fprintln(stderr, "ERROR: GjallarOS validation failed")
		return 1
	}
	return 0
}

func printUsage(out io.Writer) {
	fmt.Fprintln(out, "Usage: gjallarctl check [--repo PATH] [--timeout DURATION]")
	fmt.Fprintln(out, "       gjallarctl oddc validate-device [--repo PATH]")
	fmt.Fprintln(out, "       gjallarctl rebuild --repo PATH --host HOST [-d|--debug] [-n|--no-cleanup] [NIXOS-REBUILD-ARGS...]")
	fmt.Fprintln(out, "       gjallarctl detect {graphics|network}")
	fmt.Fprintln(out, "       gjallarctl device-probe refresh [--output PATH]")
	fmt.Fprintln(out, "       gjallarctl device-probe diagnose zbook-x2-g4 [--input PATH]")
	fmt.Fprintln(out, "       gjallarctl ai profile [--config PATH]")
	fmt.Fprintln(out, "       gjallarctl normalize keyboard --layout VALUE")
	fmt.Fprintln(out, "       gjallarctl preset {validate|get|list|bool} --config PATH [--key NAME]")
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Safe GjallarOS maintenance commands. Rebuild invokes sudo explicitly.")
}

func syncProjectToolSettings(settingsPath string, user config.User) error {
	keys := []string{
		"planeEnable",
		"planeHost",
		"drawioEnable",
		"drawioSelfHosted",
		"drawioHost",
	}

	rendered := strings.Split(string(nixrender.Render(nixrender.Settings{
		PlaneEnable:      user.PlaneEnable,
		PlaneHost:        user.PlaneHost,
		DrawioEnable:     user.DrawioEnable,
		DrawioSelfHosted: user.DrawioSelfHosted,
		DrawioHost:       user.DrawioHost,
	})), "\n")

	desired := make(map[string]string, len(keys))
	for _, line := range rendered {
		trimmed := strings.TrimSpace(line)
		for _, key := range keys {
			if strings.HasPrefix(trimmed, key+" = ") {
				desired[key] = line
			}
		}
	}

	if len(desired) != len(keys) {
		return fmt.Errorf("render project-tool settings: expected %d fields, got %d", len(keys), len(desired))
	}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", settingsPath, err)
	}

	lines := strings.Split(string(data), "\n")
	seen := make(map[string]bool, len(keys))
	result := make([]string, 0, len(lines)+len(keys))
	insertedMissing := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		replaced := false

		for _, key := range keys {
			if strings.HasPrefix(trimmed, key+" = ") {
				result = append(result, desired[key])
				seen[key] = true
				replaced = true
				break
			}
		}

		if replaced {
			continue
		}

		if !insertedMissing && strings.HasPrefix(trimmed, "backgroundNormal = ") {
			for _, key := range keys {
				if !seen[key] {
					result = append(result, desired[key])
					seen[key] = true
				}
			}
			insertedMissing = true
		}

		result = append(result, line)
	}

	for _, key := range keys {
		if !seen[key] {
			return fmt.Errorf("sync project-tool setting %s: insertion anchor not found", key)
		}
	}

	updated := strings.Join(result, "\n")
	if updated == string(data) {
		return nil
	}

	info, err := os.Stat(settingsPath)
	if err != nil {
		return fmt.Errorf("stat %s: %w", settingsPath, err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(settingsPath), ".settings-project-tools-*")
	if err != nil {
		return fmt.Errorf("create temporary settings file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if err := tmp.Chmod(info.Mode().Perm()); err != nil {
		tmp.Close()
		return fmt.Errorf("preserve settings permissions: %w", err)
	}

	if _, err := tmp.WriteString(updated); err != nil {
		tmp.Close()
		return fmt.Errorf("write temporary settings file: %w", err)
	}

	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync temporary settings file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary settings file: %w", err)
	}

	if err := os.Rename(tmpName, settingsPath); err != nil {
		return fmt.Errorf("replace %s: %w", settingsPath, err)
	}

	return nil
}

func runRebuild(args []string, stdout, stderr io.Writer) int {
	repo, host := "", ""
	debug, cleanup := false, true
	var rebuildArgs []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--repo", "--host":
			if i+1 >= len(args) {
				fmt.Fprintf(stderr, "ERROR: %s requires a value\n", args[i])
				return 2
			}
			i++
			if args[i-1] == "--repo" {
				repo = args[i]
			} else {
				host = args[i]
			}
		case "-d", "--debug":
			debug = true
		case "-n", "--no-cleanup":
			cleanup = false
		case "--":
			rebuildArgs = append(rebuildArgs, args[i+1:]...)
			i = len(args)
		default:
			rebuildArgs = append(rebuildArgs, args[i])
		}
	}
	if repo == "" || host == "" {
		fmt.Fprintln(stderr, "ERROR: rebuild requires --repo and --host")
		return 2
	}

	formattedJSON, err := repojson.CanonicalizeChangedTracked(
		context.Background(),
		repo,
	)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: canonicalize changed JSON: %v\n", err)
		return 1
	}
	for _, path := range formattedJSON {
		fmt.Fprintf(stdout, "PASS: canonical JSON: %s\n", path)
	}

	messagesPath := filepath.Join(repo, "system/tools/commands/rebuild-messages.json")
	if _, err := os.Stat(messagesPath); errors.Is(err, os.ErrNotExist) {
		messagesPath = filepath.Join(repo, "system/tools/scripts/rebuild-messages.json")
	}
	messages, err := loadRebuildMessages(messagesPath, host)
	if err != nil {
		fmt.Fprintf(stderr, "[GjallarOS] Error: %v\n", err)
		return 1
	}

	// Managed JODS endpoints deliberately remove the desktop user from
	// wheel. Rebuilds on those systems therefore cross the explicit root
	// authentication boundary instead of attempting sudo.
	configPath := filepath.Join(repo, "user.config.json")
	document, err := preset.Load(configPath)
	if err != nil {
		fmt.Fprintf(stderr, "[GjallarOS] Error: load user configuration: %v\n", err)
		return 1
	}

	projectTools := config.User{}

	projectTools.PlaneEnable, err = document.Bool("planeEnable")
	if err != nil {
		fmt.Fprintf(stderr, "[GjallarOS] Error: read planeEnable: %v\n", err)
		return 1
	}

	projectTools.PlaneHost, err = document.String("planeHost")
	if err != nil {
		fmt.Fprintf(stderr, "[GjallarOS] Error: read planeHost: %v\n", err)
		return 1
	}

	projectTools.DrawioEnable, err = document.Bool("drawioEnable")
	if err != nil {
		fmt.Fprintf(stderr, "[GjallarOS] Error: read drawioEnable: %v\n", err)
		return 1
	}

	projectTools.DrawioSelfHosted, err = document.Bool("drawioSelfHosted")
	if err != nil {
		fmt.Fprintf(stderr, "[GjallarOS] Error: read drawioSelfHosted: %v\n", err)
		return 1
	}

	projectTools.DrawioHost, err = document.String("drawioHost")
	if err != nil {
		fmt.Fprintf(stderr, "[GjallarOS] Error: read drawioHost: %v\n", err)
		return 1
	}

	if err := config.NormalizeProjectTools(&projectTools); err != nil {
		fmt.Fprintf(stderr, "[GjallarOS] Error: normalize project-tool settings: %v\n", err)
		return 1
	}

	settingsPath := filepath.Join(repo, "settings.nix")
	if err := syncProjectToolSettings(settingsPath, projectTools); err != nil {
		fmt.Fprintf(stderr, "[GjallarOS] Error: sync project-tool settings: %v\n", err)
		return 1
	}

	managedDevice, err := document.Bool("endpointManagedDevice")
	if err != nil {
		fmt.Fprintf(stderr, "[GjallarOS] Error: read endpointManagedDevice: %v\n", err)
		return 1
	}

	if managedDevice && os.Geteuid() != 0 {
		executable, err := os.Executable()
		if err != nil {
			fmt.Fprintf(stderr, "[GjallarOS] Error: resolve gjallarctl executable: %v\n", err)
			return 1
		}

		home, _ := os.UserHomeDir()

		rootArgs := []string{
			"env",
			"GJALLAR_REBUILD_CALLER_HOME=" + home,
			executable,
			"rebuild",
		}
		rootArgs = append(rootArgs, args...)

		fmt.Fprintln(
			stdout,
			"[GjallarOS] Managed endpoint: authenticating as root for rebuild.",
		)

		return runCommand(
			context.Background(),
			stdout,
			stderr,
			"su",
			"-c",
			shellJoin(rootArgs),
			"root",
		)
	}
	if os.Geteuid() != 0 {
		if status := runCommand(
			context.Background(),
			stdout,
			stderr,
			"sudo",
			"-v",
		); status != 0 {
			return status
		}
	}

	started := time.Now()
	commandArgs := []string{"nixos-rebuild", "switch", "--flake", "path:" + repo + "#" + host}
	rebuildHome := os.Getenv("GJALLAR_REBUILD_CALLER_HOME")
	if rebuildHome == "" {
		rebuildHome, _ = os.UserHomeDir()
	}

	if rebuildHome != "" {
		palette := filepath.Join(rebuildHome, ".local", "state", "noctalia", "stylix-override.json")
		if info, err := os.Stat(palette); err == nil && info.Mode().IsRegular() {
			commandArgs = append([]string{"env", "GJALLAR_NOCTALIA_PALETTE=" + palette}, commandArgs...)
			commandArgs = append(commandArgs, "--impure")
		}
	}
	if debug {
		commandArgs = append(commandArgs, "--show-trace")
	}
	commandArgs = append(commandArgs, rebuildArgs...)

	var status int
	if debug {
		fmt.Fprintf(stdout, "Rebuilding NixOS for %s\n[GjallarOS] Flake: %s#%s\n\n", host, repo, host)
		status = runPrivilegedCommand(
			context.Background(),
			stdout,
			stderr,
			commandArgs...,
		)
	} else {
		status = runRebuildQuiet(stdout, stderr, host, messages, started, commandArgs)
	}
	if status == 0 && cleanup {
		status = runCleanupOld(nil, stdout, stderr)
	}
	elapsed := int(time.Since(started).Round(time.Second) / time.Second)
	if status == 130 {
		fmt.Fprintf(stderr, "\n⏱ %02d:%02d  ✗ Rebuild cancelled.\n", elapsed/60, elapsed%60)
	} else if status == 0 {
		fmt.Fprintf(stdout, "\n⏱ %02d:%02d  ✓ Rebuild completed successfully.\n", elapsed/60, elapsed%60)
	} else {
		fmt.Fprintf(stderr, "\n⏱ %02d:%02d  ✗ Rebuild failed.\n", elapsed/60, elapsed%60)
	}
	return status
}

func runUpdate(args []string, stdout, stderr io.Writer) int {
	repo := os.Getenv("GJALLAROS_REPO")
	if repo == "" {
		repo = "."
	}
	mode := "update"
	if len(args) > 1 {
		fmt.Fprintln(stderr, "Usage: update [--rebuild|-r|--check]")
		return 2
	}
	if len(args) == 1 {
		mode = args[0]
	}
	if mode == "--check" {
		return runCommand(context.Background(), stdout, stderr, "nix", "flake", "check", repo)
	}
	if mode != "update" && mode != "--rebuild" && mode != "-r" {
		fmt.Fprintln(stderr, "Usage: update [--rebuild|-r|--check]")
		return 2
	}
	if status := runCommand(context.Background(), stdout, stderr, "nix", "flake", "update", repo); status != 0 {
		return status
	}
	if mode == "--rebuild" || mode == "-r" {
		return runCommand(context.Background(), stdout, stderr, "rebuild")
	}
	return 0
}

func runCleanup(args []string, stdout, stderr io.Writer) int {
	keep := "5"
	for i := 0; i < len(args); i++ {
		if (args[i] == "-k" || args[i] == "--keep") && i+1 < len(args) {
			i++
			keep = args[i]
		} else if args[i] == "-h" || args[i] == "--help" {
			fmt.Fprintln(stdout, "Usage: cleanup [--keep N]")
			return 0
		} else {
			fmt.Fprintln(stderr, "Usage: cleanup [--keep N]")
			return 2
		}
	}
	for _, r := range keep {
		if r < '0' || r > '9' {
			fmt.Fprintln(stderr, "Keep count must be numeric.")
			return 2
		}
	}
	for _, command := range [][]string{{"sudo", "nix-env", "--profile", "/nix/var/nix/profiles/system", "--delete-generations", "+" + keep}, {"nix", "store", "gc"}, {"sudo", "nix", "store", "gc"}} {
		if status := runCommand(context.Background(), stdout, stderr, command[0], command[1:]...); status != 0 {
			return status
		}
	}
	return 0
}

func runCleanupOld(args []string, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "Usage: cleanup-old-generations")
		return 2
	}
	var generationCommand *exec.Cmd
	if os.Geteuid() == 0 {
		generationCommand = exec.Command(
			"nix-env",
			"--profile",
			"/nix/var/nix/profiles/system",
			"--list-generations",
		)
	} else {
		generationCommand = exec.Command(
			"sudo",
			"nix-env",
			"--profile",
			"/nix/var/nix/profiles/system",
			"--list-generations",
		)
	}

	data, err := generationCommand.Output()
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}
	var generations []string
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) > 0 {
			if _, err := strconv.Atoi(fields[0]); err == nil {
				generations = append(generations, fields[0])
			}
		}
	}
	if len(generations) <= 5 {
		fmt.Fprintf(stdout, "[GjallarOS] Nothing to clean (%d generations, keeping 5).\n", len(generations))
		return 0
	}
	remove := generations[:len(generations)-5]
	fmt.Fprintf(stdout, "[GjallarOS] Removing %d old generations; keeping 5.\n", len(remove))
	command := []string{"nix-env", "--profile", "/nix/var/nix/profiles/system", "--delete-generations"}
	command = append(command, remove...)
	return runPrivilegedCommand(
		context.Background(),
		stdout,
		stderr,
		command...,
	)
}

func runThermalStatus(args []string, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		return 2
	}
	for _, item := range []struct {
		title, name string
		args        []string
	}{{"uptime", "uptime", nil}, {"sensors", "sensors", nil}, {"power profile", "powerprofilesctl", []string{"get"}}} {
		fmt.Fprintf(stdout, "\n== %s ==\n", item.title)
		_ = runCommand(context.Background(), stdout, stderr, item.name, item.args...)
	}
	fmt.Fprintln(stdout, "\n== top CPU ==")
	data, err := exec.Command("ps", "-eo", "pid,user,comm,%cpu,%mem", "--sort=-%cpu").Output()
	if err != nil {
		fmt.Fprintf(stderr, "ps: %v\n", err)
		return 1
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for line := 0; line < 20 && scanner.Scan(); line++ {
		fmt.Fprintln(stdout, scanner.Text())
	}
	return 0
}

func runThermalTest(args []string, stdout, stderr io.Writer) int {
	if len(args) != 2 {
		fmt.Fprintln(stderr, "Usage: thermal-test <off|on|status> <process-pattern>")
		return 2
	}
	data, _ := exec.Command("pgrep", "-f", args[1]).Output()
	pids := strings.Fields(string(data))
	if len(pids) == 0 {
		fmt.Fprintln(stdout, "No matching processes.")
		return 0
	}
	switch args[0] {
	case "off", "on":
		sig := syscall.SIGSTOP
		if args[0] == "on" {
			sig = syscall.SIGCONT
		}
		for _, pid := range pids {
			var n int
			fmt.Sscanf(pid, "%d", &n)
			_ = syscall.Kill(n, sig)
		}
		return 0
	case "status":
		return runCommand(context.Background(), stdout, stderr, "ps", "-o", "pid,stat,comm,args=", "-p", strings.Join(pids, ","))
	default:
		fmt.Fprintf(stderr, "Unknown action: %s\n", args[0])
		return 2
	}
}

func runHelpme(args []string, stdout, stderr io.Writer) int {
	if len(args) > 1 || len(args) == 1 && args[0] != "--text" {
		return 2
	}
	text := "GjallarOS tools\n\n  rebuild            Apply the current NixOS configuration.\n  update             Update flake inputs.\n  cleanup            Remove old generations and collect garbage.\n  thermal-status     Show temperatures and power state.\n  thermal-test       Pause/resume processes for troubleshooting.\n  check-installer    Check installer configuration.\n"
	if len(args) == 1 || os.Getenv("DISPLAY")+os.Getenv("WAYLAND_DISPLAY") == "" {
		fmt.Fprint(stdout, text)
		return 0
	}
	if _, err := exec.LookPath("yad"); err != nil {
		fmt.Fprint(stdout, text)
		return 0
	}
	return runCommand(context.Background(), stdout, stderr, "yad",
		"--list", "--title=GjallarOS tools", "--width=900", "--height=520", "--center", "--button=Close:0",
		"--column=Command", "--column=Description",
		"rebuild", "Apply the current NixOS configuration.",
		"update", "Update flake inputs.",
		"cleanup", "Remove old generations and collect garbage.",
		"thermal-status", "Show temperatures and power state.",
		"thermal-test", "Pause/resume processes for troubleshooting.",
		"check-installer", "Check installer configuration.")
}

func runRebuildQuiet(stdout, stderr io.Writer, host string, messages []string, started time.Time, commandArgs []string) int {
	log, err := os.CreateTemp("", "gjallar-rebuild-*.log")
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: create rebuild log: %v\n", err)
		return 1
	}
	defer os.Remove(log.Name())
	defer log.Close()

	terminal, ok := stdout.(*os.File)
	interactive := ok && isTerminal(terminal)
	stop := make(chan struct{})
	var ui sync.WaitGroup
	if interactive {
		fmt.Fprint(stdout, rebuildUIEnter)
		ui.Add(1)
		go func() {
			defer ui.Done()
			index, shownAt := rand.IntN(len(messages)), 0
			ticker := time.NewTicker(time.Second / 4)
			defer ticker.Stop()
			for {
				elapsed := int(time.Since(started) / time.Second)
				width := terminalWidth(terminal)
				if elapsed-shownAt >= rebuildMessageSeconds(messages[index], width) {
					if len(messages) > 1 {
						next := index
						for next == index {
							next = rand.IntN(len(messages))
						}
						index = next
					}
					shownAt = elapsed
				}
				fmt.Fprint(stdout, rebuildFrame(host, messages[index], elapsed, elapsed-shownAt, width))
				select {
				case <-stop:
					return
				case <-ticker.C:
				}
			}
		}()
	} else {
		fmt.Fprintf(stdout, "Rebuilding NixOS for %s\n", host)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	status := runPrivilegedCommand(
		ctx,
		log,
		log,
		commandArgs...,
	)
	cancel()
	if interactive {
		close(stop)
		ui.Wait()
		fmt.Fprint(stdout, rebuildUILeave)
	}
	if status != 0 {
		if _, err := log.Seek(0, io.SeekStart); err == nil {
			_, _ = io.Copy(stderr, log)
		}
	}
	return status
}

func runPrivilegedCommand(
	ctx context.Context,
	stdout, stderr io.Writer,
	args ...string,
) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "ERROR: empty privileged command")
		return 2
	}

	if os.Geteuid() == 0 {
		return runCommand(
			ctx,
			stdout,
			stderr,
			args[0],
			args[1:]...,
		)
	}

	return runCommand(
		ctx,
		stdout,
		stderr,
		"sudo",
		args...,
	)
}

func shellJoin(args []string) string {
	quoted := make([]string, len(args))

	for i, arg := range args {
		quoted[i] = shellQuote(arg)
	}

	return strings.Join(quoted, " ")
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}

	return "'" + strings.ReplaceAll(
		value,
		"'",
		"'\"'\"'",
	) + "'"
}

func runCommand(ctx context.Context, stdout, stderr io.Writer, name string, args ...string) int {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout, cmd.Stderr, cmd.Stdin = stdout, stderr, os.Stdin
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	err := cmd.Run()
	if err == nil {
		return 0
	}
	if ctx.Err() != nil && cmd.Process != nil {
		_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
		return 130
	}
	if exit, ok := err.(*exec.ExitError); ok {
		return exit.ExitCode()
	}
	fmt.Fprintf(stderr, "ERROR: %s: %v\n", name, err)
	return 1
}

func loadRebuildMessages(path, host string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read rebuild messages: %w", err)
	}
	var document any
	if err := json.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("parse rebuild messages: %w", err)
	}
	var messages []string
	var visit func(any)
	visit = func(value any) {
		switch value := value.(type) {
		case string:
			prefix := "Rebuilding NixOS for $runtime_host... "
			if strings.HasPrefix(value, prefix) {
				messages = append(messages, strings.ReplaceAll(strings.TrimPrefix(value, prefix), "$runtime_host", host))
			}
		case []any:
			for _, item := range value {
				visit(item)
			}
		case map[string]any:
			for _, item := range value {
				visit(item)
			}
		}
	}
	visit(document)
	if len(messages) == 0 {
		return nil, fmt.Errorf("no rebuild messages found in %s", path)
	}
	return messages, nil
}

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func terminalWidth(file *os.File) int {
	type winsize struct{ Row, Col, Xpixel, Ypixel uint16 }
	var size winsize
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, file.Fd(), uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(&size)))
	if errno != 0 || size.Col < 2 {
		return 80
	}
	return int(size.Col)
}

func rebuildFrame(host, message string, elapsedSeconds, messageSeconds, width int) string {
	if width < 2 {
		width = 80
	}
	prefix := fmt.Sprintf("⏱ [%02d:%02d]  ", elapsedSeconds/60, elapsedSeconds%60)
	available := width - terminalCellWidth(prefix) - 2
	if available < 1 {
		available = 1
	}
	overflow := terminalCellWidth(message) - available
	if overflow < 0 {
		overflow = 0
	}
	offset := messageSeconds * 4
	if offset > overflow {
		offset = overflow
	}
	status := prefix + terminalCellSlice(message, offset, available)
	return "\x1b[H\x1b[2J" + terminalCellSlice("Rebuilding NixOS for "+host, 0, width-1) + "\n" + status
}

func rebuildMessageSeconds(message string, width int) int {
	available := width - terminalCellWidth("⏱ [00:00]  ") - 2
	if available < 1 {
		available = 1
	}
	overflow := terminalCellWidth(message) - available
	if overflow <= 0 {
		return 5
	}
	seconds := (overflow+3)/4 + 2
	if seconds < 5 {
		return 5
	}
	return seconds
}

func terminalCellWidth(value string) int {
	width := 0
	for _, r := range value {
		width += terminalRuneWidth(r)
	}
	return width
}

func terminalCellSlice(value string, skip, limit int) string {
	if limit <= 0 {
		return ""
	}
	var out strings.Builder
	position, used := 0, 0
	for _, r := range value {
		width := terminalRuneWidth(r)
		if position+width <= skip {
			position += width
			continue
		}
		if position < skip {
			position += width
			continue
		}
		if used+width > limit {
			break
		}
		out.WriteRune(r)
		used += width
		position += width
	}
	return out.String()
}

func terminalRuneWidth(r rune) int {
	if r == utf8.RuneError || r == 0 || r == '\n' || r == '\r' || unicode.IsControl(r) {
		return 0
	}
	if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) || r == '\u200d' || r == '\ufe0f' {
		return 0
	}
	if r == 0x23f1 || r >= 0x1100 && (r <= 0x115f || r == 0x2329 || r == 0x232a ||
		(r >= 0x2e80 && r <= 0xa4cf) || (r >= 0xac00 && r <= 0xd7a3) ||
		(r >= 0xf900 && r <= 0xfaff) || (r >= 0xfe10 && r <= 0xfe19) ||
		(r >= 0xfe30 && r <= 0xfe6f) || (r >= 0xff00 && r <= 0xff60) ||
		(r >= 0xffe0 && r <= 0xffe6) || (r >= 0x1f300 && r <= 0x1faff) ||
		(r >= 0x20000 && r <= 0x3fffd)) {
		return 2
	}
	return 1
}

type hyprMonitor struct {
	Name        string  `json:"name"`
	Width       int     `json:"width"`
	Height      int     `json:"height"`
	RefreshRate float64 `json:"refreshRate"`
	X           int     `json:"x"`
	Y           int     `json:"y"`
	Scale       float64 `json:"scale"`
}

func internalDRMConnector(sysRoot string) (string, error) {
	paths, err := filepath.Glob(filepath.Join(sysRoot, "class", "drm", "card*-eDP-*"))
	if err != nil {
		return "", err
	}

	for _, path := range paths {
		status, err := os.ReadFile(filepath.Join(path, "status"))
		if err != nil {
			continue
		}
		if strings.TrimSpace(string(status)) == "connected" {
			name := filepath.Base(path)
			if i := strings.Index(name, "-eDP-"); i >= 0 {
				return name[i+1:], nil
			}
		}
	}

	return "", fmt.Errorf("no connected internal eDP connector found")
}

func hyprlandMonitors() ([]hyprMonitor, error) {
	out, err := exec.Command("hyprctl", "monitors", "-j").Output()
	if err != nil {
		return nil, fmt.Errorf("hyprctl monitors: %w", err)
	}

	var monitors []hyprMonitor
	if err := json.Unmarshal(out, &monitors); err != nil {
		return nil, fmt.Errorf("decode hyprctl monitors: %w", err)
	}

	return monitors, nil
}

func orientationTransform(value string) (int, bool) {
	switch strings.TrimSpace(value) {
	case "normal":
		return 0, true
	case "right-up":
		return 1, true
	case "bottom-up":
		return 2, true
	case "left-up":
		return 3, true
	default:
		return 0, false
	}
}

func applyHyprlandRotation(sysRoot, orientation string) error {
	transform, ok := orientationTransform(orientation)
	if !ok {
		return nil
	}

	connector, err := internalDRMConnector(sysRoot)
	if err != nil {
		return err
	}

	monitors, err := hyprlandMonitors()
	if err != nil {
		return err
	}

	var monitor *hyprMonitor
	for i := range monitors {
		if monitors[i].Name == connector {
			monitor = &monitors[i]
			break
		}
	}
	if monitor == nil {
		return fmt.Errorf("internal connector %q not present in Hyprland", connector)
	}

	mode := fmt.Sprintf("%dx%d@%.3f", monitor.Width, monitor.Height, monitor.RefreshRate)
	position := fmt.Sprintf("%dx%d", monitor.X, monitor.Y)
	scale := strconv.FormatFloat(monitor.Scale, 'f', -1, 64)
	spec := fmt.Sprintf(
		"%s,%s,%s,%s,transform,%d",
		monitor.Name,
		mode,
		position,
		scale,
		transform,
	)

	for _, args := range [][]string{
		{"keyword", "monitor", spec},
		{"keyword", "input:touchdevice:transform", strconv.Itoa(transform)},
		{"keyword", "input:tablet:transform", strconv.Itoa(transform)},
	} {
		cmd := exec.Command("hyprctl", args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf(
				"hyprctl %s: %w: %s",
				strings.Join(args, " "),
				err,
				strings.TrimSpace(string(out)),
			)
		}
	}

	return nil
}

func runHyprlandRotate(args []string, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "usage: gjallarctl hyprland-rotate")
		return 2
	}

	cmd := exec.Command("monitor-sensor")
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		fmt.Fprintf(stderr, "FAIL: monitor-sensor stdout: %v\n", err)
		return 1
	}
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(stderr, "FAIL: start monitor-sensor: %v\n", err)
		return 1
	}

	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()

		var orientation string
		switch {
		case strings.Contains(line, "Accelerometer orientation changed:"):
			orientation = strings.TrimSpace(
				strings.TrimPrefix(
					line[strings.Index(line, "Accelerometer orientation changed:"):],
					"Accelerometer orientation changed:",
				),
			)

		case strings.Contains(line, "Has accelerometer (orientation:"):
			start := strings.Index(line, "orientation:")
			if start >= 0 {
				rest := line[start+len("orientation:"):]
				if end := strings.IndexAny(rest, ",)"); end >= 0 {
					rest = rest[:end]
				}
				orientation = strings.TrimSpace(rest)
			}
		}

		if orientation == "" {
			continue
		}

		if err := applyHyprlandRotation("/sys", orientation); err != nil {
			fmt.Fprintf(stderr, "WARN: rotate %s: %v\n", orientation, err)
			continue
		}

		fmt.Fprintf(stdout, "PASS: orientation %s applied\n", orientation)
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(stderr, "FAIL: read monitor-sensor: %v\n", err)
		return 1
	}

	if err := cmd.Wait(); err != nil {
		fmt.Fprintf(stderr, "FAIL: monitor-sensor: %v\n", err)
		return 1
	}

	return 0
}
