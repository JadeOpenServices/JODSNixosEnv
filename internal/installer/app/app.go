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
	"github.com/bakanura/gjallarOS/internal/installer/hardwareconfig"
	"github.com/bakanura/gjallarOS/internal/installer/localgit"
	"github.com/bakanura/gjallarOS/internal/installer/nixrender"
	"github.com/bakanura/gjallarOS/internal/installer/prompt"
	"github.com/bakanura/gjallarOS/internal/installer/release"
	"github.com/bakanura/gjallarOS/internal/installer/secrets"
	"github.com/bakanura/gjallarOS/internal/installer/secureboot"
	"github.com/bakanura/gjallarOS/internal/installercheck"
)

type options struct {
	repo                                     string
	skipHardware, refreshHardware, noRebuild bool
}
type state struct {
	user             config.User
	render           nixrender.Settings
	passthroughIDs   []string
	preset, existing bool
	control          string
}

func Run(ctx context.Context, args []string, in io.Reader, out, errOut io.Writer) int {
	f := flag.NewFlagSet("gjallar-installer", flag.ContinueOnError)
	f.SetOutput(errOut)
	cwd, _ := os.Getwd()
	opt := options{}
	f.StringVar(&opt.repo, "repo", cwd, "GjallarOS repository")
	f.BoolVar(&opt.skipHardware, "skip-hardware", false, "skip hardware generation")
	f.BoolVar(&opt.refreshHardware, "refresh-hardware", false, "regenerate hardware configuration")
	f.BoolVar(&opt.noRebuild, "no-rebuild", false, "do not install a boot generation")
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
	s.existing = existingInstall(root)
	if code := prepareHost(ctx, ui, opt, s, out, errOut); code != 0 {
		return code
	}
	hardware := discovery.DetectHardware("/sys")
	choices, err := discovery.Discover(root, s.preset, hardware)
	if err != nil {
		return fail(errOut, err)
	}
	if s.preset {
		normalizePreset(&s.user, root)
	} else if err := collectInteractive(ctx, ui, root, hardware, choices, &s.user); err != nil {
		return fail(errOut, err)
	}
	if s.user.FrameworkEnable && s.user.SecureBootPrompt && !s.user.EndpointManagedDevice {
		s.user.SecureBootEnable, err = ui.Confirm(ctx, "Prepare Framework Secure Boot and recovery keys?", false)
		if err != nil {
			return fail(errOut, err)
		}
	}
	if err := validateSelections(s.user, choices); err != nil {
		return fail(errOut, err)
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
	if s.render.WorkUserEnable {
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
	settingsPath := filepath.Join(root, "settings.nix")
	if err := nixrender.WriteAtomic(settingsPath, s.render); err != nil {
		return fail(errOut, err)
	}
	fmt.Fprintln(out, "Wrote", settingsPath)
	hardwarePath := filepath.Join(root, "profiles", s.user.Profile, "hardware-configuration.nix")
	_, hardwareErr := os.Stat(hardwarePath)
	skip := opt.skipHardware || (s.existing && !opt.refreshHardware && hardwareErr == nil)
	if !skip {
		fmt.Fprintln(out, "PLAN: generate and atomically replace", hardwarePath)
		if backup, err := hardwareconfig.Generate(ctx, root, hardwarePath, time.Now()); err != nil {
			return fail(errOut, err)
		} else if backup != "" {
			fmt.Fprintln(out, "Backup:", backup)
		}
		tpmOut, err := controlOutput(ctx, s.control, errOut, "installer", "tpm2", "--repo", root, "--hardware", hardwarePath)
		if err != nil {
			return fail(errOut, err)
		}
		fmt.Fprint(out, tpmOut)
		s.render.LUKSTPM2Enable = strings.Contains(tpmOut, "luks_tpm2_enable=true")
		if err := nixrender.WriteAtomic(settingsPath, s.render); err != nil {
			return fail(errOut, err)
		}
		if err := controlAttached(ctx, s.control, "installer", "luks", "--repo", root, "--hardware", hardwarePath); err != nil {
			return fail(errOut, err)
		}
	}
	fmt.Fprintln(out, "PLAN: atomically update local Git excludes and protect generated machine configuration")
	if _, err := localgit.Protect(ctx, root, func() string {
		if skip {
			return ""
		}
		return hardwarePath
	}()); err != nil {
		return fail(errOut, err)
	}
	if s.render.SecureBootEnable && !s.render.EndpointManagedDevice {
		recovery, err := secureboot.Provision(ctx)
		if err != nil {
			return fail(errOut, err)
		}
		fmt.Fprintf(out, "\nSECURE BOOT RECOVERY ARCHIVE\n%s\n\nSECURE BOOT ARCHIVE PASSPHRASE\n%s\nIMPORTANT: store both offline. This output is not added to shell history.\n", recovery.ArchivePath, recovery.Passphrase)
		yes, err := ui.Confirm(ctx, "Have you saved the Secure Boot recovery archive and passphrase?", false)
		if err != nil || !yes {
			return fail(errOut, errors.New("Secure Boot recovery material was not confirmed saved"))
		}
		yes, err = ui.Confirm(ctx, "Are you absolutely sure the Secure Boot recovery material is saved offline?", false)
		if err != nil || !yes {
			return fail(errOut, errors.New("Secure Boot recovery material was not confirmed twice"))
		}
		recovery.Passphrase = ""
	}
	runRebuild := s.user.RunRebuild && !opt.noRebuild
	if !s.preset && !opt.noRebuild {
		runRebuild, err = ui.Confirm(ctx, "Install the next NixOS boot generation now?", false)
		if err != nil {
			return fail(errOut, err)
		}
	}
	if runRebuild {
		target, err := deploy.Target(root, s.user.Hostname)
		if err != nil {
			return fail(errOut, err)
		}
		fmt.Fprintln(out, "PLAN: validate then install", target)
		if err := deploy.Apply(ctx, target); err != nil {
			return fail(errOut, err)
		}
		if s.user.AutoReboot {
			_ = attached(ctx, "sudo", "systemctl", "reboot")
		} else if yes, _ := ui.Confirm(ctx, "Deployment complete. Reboot now?", false); yes {
			_ = attached(ctx, "sudo", "systemctl", "reboot")
		}
	}
	fmt.Fprintln(out, "GjallarOS installation complete.")
	return 0
}

func prepareHost(ctx context.Context, ui prompt.UI, opt options, s state, out, errOut io.Writer) int {
	expected, actual, err := release.Inspect(opt.repo, "/etc/os-release")
	if err != nil {
		return fail(errOut, err)
	}
	if expected != actual {
		fmt.Fprintf(out, "NixOS %s detected; target is %s.\n", actual, expected)
		fmt.Fprintf(out, "PLAN: sudo nix-channel --add https://channels.nixos.org/nixos-%s nixos\nPLAN: sudo nix-channel --update nixos\nPLAN: sudo nixos-rebuild switch --upgrade\n", expected)
		yes, err := ui.Confirm(ctx, "Align the NixOS channel and rebuild?", false)
		if err != nil || !yes {
			return fail(errOut, errors.New("release mismatch not approved"))
		}
		if err := release.Align(ctx, expected); err != nil {
			return fail(errOut, err)
		}
		_, active, inspectErr := release.Inspect(opt.repo, "/run/current-system/etc/os-release")
		if inspectErr != nil || active != expected {
			return fail(errOut, fmt.Errorf("rebuild completed, but NixOS %s is not active", expected))
		}
	}
	configPath := "/etc/nixos/configuration.nix"
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fail(errOut, err)
	}
	plan := bootstrap.Build(data, s.preset)
	if len(plan.Missing) > 0 {
		fmt.Fprintf(out, "Missing helpers/options: %s\nProposed configuration:\n%s\n", strings.Join(plan.Missing, ", "), plan.Updated)
		yes, err := ui.Confirm(ctx, "Apply prerequisite configuration and rebuild?", false)
		if err != nil {
			return fail(errOut, err)
		}
		if !yes {
			return fail(errOut, errors.New("required persistent prerequisites were not approved"))
		}
		if _, err := bootstrap.Apply(ctx, configPath, plan.Updated, time.Now()); err != nil {
			return fail(errOut, err)
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
	if u.Profile, err = ui.Choice(ctx, "Profile", first(o.Profiles), o.Profiles); err != nil {
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
	u.NemuEnable, err = ui.Confirm(ctx, "Enable Nemu virtual machines?", false)
	if err != nil {
		return err
	}
	if hardware.LaptopVendor == "framework" || strings.HasPrefix(u.Profile, "framework") {
		u.FrameworkEnable = true
		u.FrameworkModel, err = ui.Choice(ctx, "Framework model", "13", []string{"13", "16", "12"})
		if err != nil {
			return err
		}
	}
	u.EnableScrobbling, err = ui.Confirm(ctx, "Enable Last.fm and/or ListenBrainz scrobbling?", false)
	if err != nil {
		return err
	}
	if u.EnableScrobbling {
		u.EnableLastfm, err = ui.Confirm(ctx, "Enable Last.fm?", false)
		if err != nil {
			return err
		}
		u.EnableListenbrainz, err = ui.Confirm(ctx, "Enable ListenBrainz?", false)
		if err != nil {
			return err
		}
		u.EnableScrobbling = u.EnableLastfm || u.EnableListenbrainz
	}
	u.WriteConfig = true
	u.RunRebuild = true
	return nil
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

func detectAndRenderState(ctx context.Context, root string, s *state) error {
	u := s.user
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
	s.render = nixrender.Settings{System: u.System, Profile: u.Profile, Hostname: u.Hostname, Username: u.Username, Timezone: u.Timezone, Locale: u.Locale, KeyboardLayout: u.KeyboardLayout, KeyboardVariant: u.KeyboardVariant, TouchpadWorkspaceSwipe: u.TouchpadWorkspaceSwipe, ClamshellEnable: u.ClamshellEnable, USBGuardEnable: u.USBGuardEnable, Name: u.Name, Email: u.Email, GitHubUsername: u.GitHubUsername, DotfilesDir: u.DotfilesDir, WorkUserEnable: u.WorkUserEnable, WorkUsername: u.WorkUsername, DockerEnable: u.DockerEnable, DebugFunctions: u.DebugFunctions, Shell: u.Shell, Editors: u.Editors, Browsers: u.Browsers, PreferredEditor: u.PreferredEditor, PreferredBrowser: u.PreferredBrowser, BackgroundNormal: u.BackgroundNormal, BackgroundWork: u.BackgroundWork, BackgroundGaming: u.BackgroundGaming, EnableScrobbling: u.EnableScrobbling, EnableLastfm: u.EnableLastfm, EnableListenbrainz: u.EnableListenbrainz, LastfmUsername: u.LastfmUsername, ListenbrainzUsername: u.ListenbrainzUsername, FrameworkEnable: u.FrameworkEnable, FrameworkModel: u.FrameworkModel, GraphicsVendor: g.Vendor, GraphicsType: g.Type, GraphicsCompute: g.Compute, GraphicsBusID: g.BusID, GraphicsIntegratedBusID: g.IntegratedBusID, WiFiDriver: wifi, AIEnable: u.AIEnable, AIModel: ai.Model, AIContextTokens: ai.ContextTokens, AIVRAMMB: ai.VRAMMB, NemuEnable: u.NemuEnable, NemuGPUPassthrough: passthrough, LUKSTPM2Enable: false, RecoveryEnable: u.RecoveryEnable, JODSPrebootLockEnable: u.JODSPrebootLockEnable, SecureBootEnable: u.SecureBootEnable, EndpointManagedDevice: u.EndpointManagedDevice, WMs: []string{"hyprland"}, Theme: u.Theme}
	if passthrough {
		s.render.NemuGPUIDs = g.PassthroughIDs
	}
	return nil
}

func configureSecrets(ctx context.Context, root string, s *state, errOut io.Writer) error {
	if !s.user.EnableScrobbling {
		return nil
	}
	args := []string{"installer", "configure-scrobbling", "--repo", root, "--username", s.user.Username, fmt.Sprintf("--lastfm=%t", s.user.EnableLastfm), fmt.Sprintf("--listenbrainz=%t", s.user.EnableListenbrainz)}
	result, err := controlOutput(ctx, s.control, errOut, args...)
	if err != nil {
		return err
	}
	for _, line := range strings.Split(result, "\n") {
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		switch key {
		case "lastfm_username":
			s.render.LastfmUsername = value
		case "listenbrainz_username":
			s.render.ListenbrainzUsername = value
		}
	}
	_, err = secrets.Check(ctx, root, s.user.Username)
	return err
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
