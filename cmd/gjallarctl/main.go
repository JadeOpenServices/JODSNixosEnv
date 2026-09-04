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
	"io"
	"math/rand/v2"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode"
	"unicode/utf8"
	"unsafe"

	"github.com/bakanura/gjallarOS/internal/ai/profile"
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
	"github.com/bakanura/gjallarOS/internal/installer/policy"
	"github.com/bakanura/gjallarOS/internal/installer/release"
	"github.com/bakanura/gjallarOS/internal/installer/secrets"
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
	case "detect":
		return runDetect(args[1:], stdout, stderr)
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
	mapping, err := diskcrypto.Mapping(resolved)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}
	if mapping == "" {
		fmt.Fprintln(stdout, "luks_tpm2_enable=false\n[NOTE] No LUKS mapping found for TPM2 enrollment.")
		return 0
	}
	if !diskcrypto.TPMAvailable() {
		fmt.Fprintln(stdout, "luks_tpm2_enable=false\n[NOTE] No TPM2 device detected; keeping passphrase unlock.")
		return 0
	}
	device, err := diskcrypto.DetectDevice(context.Background())
	if err != nil || device == "" {
		fmt.Fprintln(stdout, "luks_tpm2_enable=false\n[NOTE] No active crypto_LUKS device found; skipping TPM2 enrollment.")
		return 0
	}
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: TPM2 enrollment requires a terminal: %v\n", err)
		return 1
	}
	defer tty.Close()
	reader := bufio.NewReader(tty)
	yes, _, err := ttyConfirm(tty, reader, "Enable TPM2 automatic unlock? The passphrase remains as recovery.")
	if err != nil || !yes {
		fmt.Fprintln(stdout, "luks_tpm2_enable=false")
		return 0
	}
	passphrase, err := workpassword.ReadSecret(tty, reader, tty, "Current LUKS passphrase for TPM enrollment: ")
	if err != nil || passphrase == "" {
		fmt.Fprintln(stderr, "WARN: no passphrase supplied; skipping TPM2 enrollment")
		return 0
	}
	fmt.Fprintf(stdout, "PLAN: sudo systemd-cryptenroll --unlock-key-file=<temporary> --tpm2-device=auto %s\n", device)
	if err := diskcrypto.EnrollTPM(context.Background(), device, passphrase); err != nil {
		passphrase = ""
		fmt.Fprintf(stderr, "ERROR: TPM2 enrollment failed; boot configuration unchanged: %v\n", err)
		return 1
	}
	passphrase = ""
	if err := diskcrypto.EnableTPMConfig(resolved, mapping); err != nil {
		fmt.Fprintf(stderr, "ERROR: TPM enrolled but hardware configuration update failed: %v\n", err)
		return 1
	}
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
	mapping, err := diskcrypto.Mapping(resolved)
	if err != nil || mapping == "" {
		fmt.Fprintln(stdout, "[NOTE] No LUKS mapping exists; skipping LUKS configuration.")
		return 0
	}
	device, err := diskcrypto.DetectDevice(context.Background())
	if err != nil || device == "" {
		fmt.Fprintln(stdout, "[NOTE] No LUKS-encrypted device detected; skipping LUKS configuration.")
		return 0
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
	fmt.Fprintln(stdout, "PLAN: enroll new key; enroll recovery key; remove original key only after both enrollments succeed")
	if err := diskcrypto.Rotate(context.Background(), device, old, newKey, recovery); err != nil {
		old = ""
		newKey = ""
		recovery = ""
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}
	old = ""
	newKey = ""
	recovery = ""
	fmt.Fprintln(stdout, "LUKS rotation complete: new and recovery keys enrolled; original key removed.")
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
	fmt.Fprintf(stdout, "PLAN: sudo nix-channel --add https://channels.nixos.org/nixos-%s nixos\nPLAN: sudo nix-channel --update nixos\nPLAN: sudo nixos-rebuild switch --upgrade\n", expected)
	if !*apply {
		return 3
	}
	if err := release.Align(context.Background(), expected); err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}
	_, active, err := release.Inspect(*repo, "/run/current-system/etc/os-release")
	if err != nil || active != expected {
		fmt.Fprintf(stderr, "ERROR: rebuild completed, but NixOS %s is not active\n", expected)
		return 1
	}
	fmt.Fprintf(stdout, "NixOS release verified: %s\n", active)
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
	f.StringVar(&s.GraphicsVendor, "graphics-vendor", "", "")
	f.StringVar(&s.GraphicsType, "graphics-type", "", "")
	f.BoolVar(&s.GraphicsCompute, "graphics-compute", false, "")
	f.StringVar(&s.GraphicsBusID, "graphics-bus-id", "", "")
	f.StringVar(&s.GraphicsIntegratedBusID, "graphics-integrated-bus-id", "", "")
	f.StringVar(&s.WiFiDriver, "wifi-driver", "", "")
	f.BoolVar(&s.AIEnable, "ai-enable", false, "")
	f.StringVar(&s.AIModel, "ai-model", "", "")
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

func runAI(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "profile" {
		fmt.Fprintln(stderr, "Usage: gjallarctl ai profile [--config PATH]")
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
	fmt.Fprintf(stdout, "profile=%s\nmodel=%s\ncontext_tokens=%d\nvram_mb=%d\nram_gb=%d\ngpu_vendor=%s\ngpu_type=%s\n",
		result.Profile, result.Model, result.ContextTokens, result.VRAMMB, result.RAMGB, result.GPUVendor, result.GPUType)
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
	fmt.Fprintln(out, "Usage: gjallarctl check [--repo PATH] [--timeout DURATION]\n       gjallarctl rebuild --repo PATH --host HOST [-d|--debug] [-n|--no-cleanup] [NIXOS-REBUILD-ARGS...]\n       gjallarctl detect {graphics|network}\n       gjallarctl ai profile [--config PATH]\n       gjallarctl normalize keyboard --layout VALUE\n       gjallarctl preset {validate|get|list|bool} --config PATH [--key NAME]")
	fmt.Fprintln(out, "\nSafe GjallarOS maintenance commands. Rebuild invokes sudo explicitly.")
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

	messagesPath := filepath.Join(repo, "system/tools/commands/rebuild-messages.json")
	if _, err := os.Stat(messagesPath); errors.Is(err, os.ErrNotExist) {
		messagesPath = filepath.Join(repo, "system/tools/scripts/rebuild-messages.json")
	}
	messages, err := loadRebuildMessages(messagesPath, host)
	if err != nil {
		fmt.Fprintf(stderr, "[GjallarOS] Error: %v\n", err)
		return 1
	}
	if status := runCommand(context.Background(), stdout, stderr, "sudo", "-v"); status != 0 {
		return status
	}

	started := time.Now()
	commandArgs := []string{"nixos-rebuild", "switch", "--flake", repo + "#" + host}
	if debug {
		commandArgs = append(commandArgs, "--show-trace")
	}
	commandArgs = append(commandArgs, rebuildArgs...)

	var status int
	if debug {
		fmt.Fprintf(stdout, "Rebuilding NixOS for %s\n[GjallarOS] Flake: %s#%s\n\n", host, repo, host)
		status = runCommand(context.Background(), stdout, stderr, "sudo", commandArgs...)
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
	cmd := exec.Command("sudo", "nix-env", "--profile", "/nix/var/nix/profiles/system", "--list-generations")
	data, err := cmd.Output()
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
	return runCommand(context.Background(), stdout, stderr, "sudo", command...)
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
	status := runCommand(ctx, log, log, "sudo", commandArgs...)
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
