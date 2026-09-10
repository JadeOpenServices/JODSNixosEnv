package prompt

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"unicode/utf8"
)

var ErrCancelled = errors.New("prompt cancelled")

type UI struct {
	Reader *bufio.Reader
	Out    io.Writer
	GTK    bool
}

func zenityCommand(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "zenity", args...)

	// The bootstrap/base NixOS environment may expose a locale which GTK
	// cannot use. Zenity then falls back to plain C and exits 255 as soon
	// as UTF-8 text such as bullets or em-dashes is rendered.
	//
	// Preserve the graphical/session environment, but replace LANG and
	// LC_ALL with a known UTF-8 locale.
	env := make([]string, 0, len(os.Environ())+2)
	for _, entry := range os.Environ() {
		if strings.HasPrefix(entry, "LANG=") ||
			strings.HasPrefix(entry, "LC_ALL=") {
			continue
		}
		env = append(env, entry)
	}

	cmd.Env = append(
		env,
		"LANG=C.UTF-8",
		"LC_ALL=C.UTF-8",
	)

	return cmd
}

func gtkAvailable() bool {
	_, err := exec.LookPath("zenity")
	return err == nil &&
		(os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != "") &&
		os.Getenv("SSH_CONNECTION") == ""
}

func gtkConfirmationWidth(message string) int {
	normalized := strings.ReplaceAll(message, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")

	totalRunes := 0
	longestParagraph := 0

	for _, paragraph := range strings.Split(normalized, "\n\n") {
		length := utf8.RuneCountInString(strings.TrimSpace(paragraph))
		totalRunes += length
		if length > longestParagraph {
			longestParagraph = length
		}
	}

	switch {
	case totalRunes <= 90 && longestParagraph <= 90:
		return 420
	case totalRunes <= 240 && longestParagraph <= 180:
		return 520
	case totalRunes <= 600 && longestParagraph <= 400:
		return 680
	default:
		return 760
	}
}

func New(in io.Reader, out io.Writer) UI {
	return UI{bufio.NewReader(in), out, gtkAvailable()}
}

func (u UI) secureTextDialog(
	ctx context.Context,
	title string,
	text string,
	okLabel string,
) error {
	if !u.GTK {
		fmt.Fprintln(u.Out)
		fmt.Fprintln(u.Out, text)
		fmt.Fprintln(u.Out)
		return nil
	}

	tmp, err := os.CreateTemp("", "gjallar-dialog-*.txt")
	if err != nil {
		return fmt.Errorf("create GTK text file: %w", err)
	}

	name := tmp.Name()
	defer os.Remove(name)

	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return fmt.Errorf("protect GTK text file: %w", err)
	}

	if _, err := tmp.WriteString(text); err != nil {
		tmp.Close()
		return fmt.Errorf("write GTK text file: %w", err)
	}

	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync GTK text file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close GTK text file: %w", err)
	}

	cmd := zenityCommand(
		ctx,
		"--text-info",
		"--title="+title,
		"--width=760",
		"--height=680",
		"--filename="+name,
		"--ok-label="+okLabel,
	)

	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())

		if exit, ok := err.(*exec.ExitError); ok {
			if msg == "" {
				msg = err.Error()
			}

			return fmt.Errorf(
				"GTK dialog %q failed with exit %d: %s",
				title,
				exit.ExitCode(),
				msg,
			)
		}

		return fmt.Errorf("launch GTK dialog %q: %w", title, err)
	}

	return nil
}

func (u UI) Confirm(ctx context.Context, message string, defaultYes bool) (bool, error) {
	if u.GTK || gtkAvailable() {
		width := gtkConfirmationWidth(message)
		args := []string{
			"--question",
			"--title=GjallarOS installer",
			"--text=" + message,
			"--width=" + strconv.Itoa(width),
			"--ok-label=Yes",
			"--cancel-label=No",
		}
		err := zenityCommand(ctx, args...).Run()
		if err == nil {
			return true, nil
		}
		if exit, ok := err.(*exec.ExitError); ok && exit.ExitCode() == 1 {
			return false, nil
		}
		return false, ErrCancelled
	}
	suffix := " [y/N] "
	if defaultYes {
		suffix = " [Y/n] "
	}
	fmt.Fprint(u.Out, message+suffix)
	line, err := u.Reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	value := strings.ToLower(strings.TrimSpace(line))
	if value == "" {
		return defaultYes, nil
	}
	return value == "y" || value == "yes", nil
}

// SecureBootFirmwareHandoff presents the security-sensitive firmware
// ownership-transfer instructions as GTK dialogs when GTK/Zenity is
// available. The user must explicitly understand the instructions and then
// decline another review before the installer may reboot into firmware.
// ShowSecureBootRecovery displays the generated recovery material before
// the installer asks the user to confirm that it has been saved.
//
// In GTK mode the secret is passed to Zenity through a mode-0600 temporary
// file instead of command-line arguments, avoiding exposure through argv.
func (u UI) ShowSecureBootRecovery(
	ctx context.Context,
	archivePath string,
	passphrase string,
) error {
	if archivePath == "" || passphrase == "" {
		return fmt.Errorf("Secure Boot recovery material is incomplete")
	}

	text := fmt.Sprintf(`GJALLAROS SECURE BOOT RECOVERY MATERIAL

SAVE BOTH ITEMS BELOW BEFORE CONTINUING.


SECURE BOOT RECOVERY ARCHIVE

%s


SECURE BOOT RECOVERY PASSPHRASE

%s


IMPORTANT

The encrypted archive contains this machine's GjallarOS Secure Boot
signing keys.

The recovery passphrase is required to decrypt that archive.

Store BOTH items somewhere safe and offline.

Do not continue until you have saved both.
`, archivePath, passphrase)

	return u.secureTextDialog(
		ctx,
		"GjallarOS Secure Boot Recovery Key",
		text,
		"I have saved this recovery material",
	)
}

func (u UI) SecureBootEnableHandoff(
	ctx context.Context,
	firmwareName string,
) error {
	firmwareName = strings.TrimSpace(firmwareName)
	if firmwareName == "" {
		firmwareName = "system firmware"
	}

	instructions := fmt.Sprintf(`GJALLAROS SECURE BOOT
FINAL FIRMWARE ACTION

GjallarOS ownership is already enrolled and verified.

Firmware: %s

1. Enable Secure Boot.
2. Save the firmware configuration.
3. Boot GjallarOS.

Do not clear, erase, reset, or replace Secure Boot keys during this final step.
`, firmwareName)

	if err := u.secureTextDialog(
		ctx,
		"GjallarOS Secure Boot - Enable Secure Boot",
		instructions,
		"Continue",
	); err != nil {
		return err
	}

	understood, err := u.Confirm(
		ctx,
		"Enable Secure Boot without clearing or replacing any keys?",
		false,
	)
	if err != nil {
		return err
	}
	if !understood {
		return fmt.Errorf("Secure Boot enable instructions were not acknowledged")
	}

	return nil
}

func (u UI) SecureBootFirmwareHandoff(
	ctx context.Context,
	firmwareName string,
	policyInstructions []string,
) error {
	firmwareName = strings.TrimSpace(firmwareName)
	if firmwareName == "" {
		firmwareName = "system firmware"
	}

	clean := make([]string, 0, len(policyInstructions))
	for _, instruction := range policyInstructions {
		instruction = strings.TrimSpace(instruction)
		if instruction != "" {
			clean = append(clean, instruction)
		}
	}

	if len(clean) == 0 {
		return fmt.Errorf("Secure Boot firmware policy contains no display instructions")
	}

	instructions := fmt.Sprintf(`GJALLAROS SECURE BOOT
FIRMWARE ACTION REQUIRED

Firmware: %s

Follow the detected device firmware policy exactly:

%s

After completing the firmware action:

- save the firmware configuration
- boot GjallarOS normally
- leave further Secure Boot changes to the GjallarOS enrollment transaction

GjallarOS will verify the expected firmware state before performing enrollment.
`,
		firmwareName,
		strings.Join(clean, "\n"),
	)

	explanation := `WHY THIS IS DEVICE-SPECIFIC

Different firmware implementations require different steps to enter a safe
Secure Boot enrollment state.

GjallarOS therefore does not hardcode a vendor procedure in the installer UI.
The instructions above come from the detected device's validated ODDC firmware
policy.

The policy is descriptive data only. It cannot provide commands, scripts,
executables, URLs, or arbitrary shell input.

GjallarOS will independently verify the resulting firmware state before any
key enrollment is allowed.
`

	for {
		if err := u.secureTextDialog(
			ctx,
			"GjallarOS Secure Boot - Firmware Instructions",
			instructions,
			"Continue",
		); err != nil {
			return err
		}

		understood, err := u.Confirm(
			ctx,
			"Do you understand the Secure Boot firmware instructions?",
			false,
		)
		if err != nil {
			return err
		}

		if !understood {
			if err := u.secureTextDialog(
				ctx,
				"GjallarOS Secure Boot - Explanation",
				explanation,
				"Back to instructions",
			); err != nil {
				return err
			}

			continue
		}

		showAgain, err := u.Confirm(
			ctx,
			"Would you like to see the previous instructions again?",
			false,
		)
		if err != nil {
			return err
		}

		if showAgain {
			continue
		}

		return nil
	}
}

func (u UI) Exact(ctx context.Context, label, expected string) error {
	if expected == "" {
		return fmt.Errorf("exact confirmation value must not be empty")
	}

	var value string

	if u.GTK || gtkAvailable() {
		out, err := zenityCommand(
			ctx,
			"--entry",
			"--title=GjallarOS installer",
			"--text="+label,
		).Output()
		if err != nil {
			return ErrCancelled
		}
		value = strings.TrimSuffix(string(out), "\n")
		value = strings.TrimSuffix(value, "\r")
	} else {
		fmt.Fprintf(u.Out, "%s: ", label)
		line, err := u.Reader.ReadString('\n')
		if err != nil {
			return err
		}
		value = strings.TrimSuffix(line, "\n")
		value = strings.TrimSuffix(value, "\r")
	}

	if value != expected {
		return fmt.Errorf("destructive confirmation did not match exactly")
	}

	return nil
}

func (u UI) Value(ctx context.Context, label, def string) (string, error) {
	if u.GTK || gtkAvailable() {
		out, err := zenityCommand(ctx, "--entry", "--title=GjallarOS installer", "--text="+label, "--entry-text="+def).Output()
		if err != nil {
			return "", ErrCancelled
		}
		return strings.TrimSpace(string(out)), nil
	}
	fmt.Fprintf(u.Out, "%s [%s]: ", label, def)
	line, err := u.Reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		line = def
	}
	return line, nil
}

func (u UI) Choice(ctx context.Context, label, def string, options []string) (string, error) {
	if len(options) == 0 {
		return "", fmt.Errorf("no options for %s", label)
	}
	if u.GTK || gtkAvailable() {
		args := []string{"--list", "--radiolist", "--title=GjallarOS installer", "--text=" + label, "--column=Selected", "--column=Value"}
		for _, v := range options {
			args = append(args, strconv.FormatBool(v == def), v)
		}
		out, err := zenityCommand(ctx, args...).Output()
		if err != nil {
			return "", ErrCancelled
		}
		value := strings.TrimSpace(string(out))
		if value == "" {
			return "", ErrCancelled
		}
		return value, nil
	}
	for i, v := range options {
		fmt.Fprintf(u.Out, "  %d) %s\n", i+1, v)
	}
	for {
		value, err := u.Value(ctx, label, def)
		if err != nil {
			return "", err
		}
		for _, candidate := range options {
			if value == candidate {
				return value, nil
			}
		}
		if n, e := strconv.Atoi(value); e == nil && n > 0 && n <= len(options) {
			return options[n-1], nil
		}
		fmt.Fprintln(u.Out, "Choose one listed value.")
	}
}

func (u UI) Multi(ctx context.Context, label string, defaults, options []string) ([]string, error) {
	selected := map[string]bool{}
	for _, v := range defaults {
		selected[v] = true
	}
	if u.GTK || gtkAvailable() {
		args := []string{"--list", "--checklist", "--title=GjallarOS installer", "--text=" + label, "--separator=\n", "--column=Selected", "--column=Value"}
		for _, v := range options {
			args = append(args, strconv.FormatBool(selected[v]), v)
		}
		out, err := zenityCommand(ctx, args...).Output()
		if err != nil {
			return nil, ErrCancelled
		}
		values := strings.Fields(string(out))
		if len(values) == 0 {
			return nil, fmt.Errorf("select at least one value")
		}
		return values, nil
	}
	fmt.Fprintf(u.Out, "%s (comma-separated) [%s]: ", label, strings.Join(defaults, ","))
	line, err := u.Reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return defaults, nil
	}
	allowed := map[string]bool{}
	for _, v := range options {
		allowed[v] = true
	}
	values := []string{}
	for _, v := range strings.Split(line, ",") {
		v = strings.TrimSpace(v)
		if !allowed[v] {
			return nil, fmt.Errorf("unsupported selection %q", v)
		}
		values = append(values, v)
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("select at least one value")
	}
	return values, nil
}
