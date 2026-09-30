package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bakanura/gjallarOS/internal/usbtrust"
	"github.com/bakanura/gjallarOS/internal/usbtrust/broker"
)

func runUSB(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "Usage: gjallarctl usb {status|audit|policy|review|provision-key|allow-once|trust-permanent|keep-blocked|enroll-internal|accept-replacement|forget} [--runtime-id ID --connection TOKEN] [--trusted-id ID] [--role ROLE] [--portable=true|false]")
		return 2
	}
	if args[0] == "review" {
		return runUSBReview(args[1:], stderr)
	}
	f := flag.NewFlagSet("usb", flag.ContinueOnError)
	f.SetOutput(stderr)
	r := broker.Request{Action: broker.Action(args[0])}
	socket := f.String("socket", broker.DefaultSocket, "USB trust broker socket")
	f.StringVar(&r.RuntimeID, "runtime-id", "", "current runtime ID from policy")
	f.StringVar(&r.Connection, "connection", "", "review connection token from policy")
	f.StringVar(&r.TrustedID, "trusted-id", "", "accepted trust record ID")
	f.StringVar(&r.Role, "role", "", "ODDC expectation role")
	portable := f.String("portable", "", "explicit true or false for permanent external trust")
	if err := f.Parse(args[1:]); err != nil {
		return 2
	}
	if f.NArg() != 0 {
		fmt.Fprintln(stderr, "FAIL: unexpected USB command arguments")
		return 2
	}
	if *portable != "" {
		value, err := strconv.ParseBool(*portable)
		if err != nil {
			fmt.Fprintln(stderr, "FAIL: --portable must be true or false")
			return 2
		}
		r.Portable = &value
	}
	if err := broker.ValidateRequest(r); err != nil {
		fmt.Fprintf(stderr, "FAIL: %v\n", err)
		return 2
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	response, err := broker.Call(ctx, *socket, r)
	if err != nil {
		fmt.Fprintf(stderr, "FAIL: %v\n", err)
		return 1
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(response); err != nil {
		fmt.Fprintf(stderr, "FAIL: %v\n", err)
		return 1
	}
	return 0
}

// Review presents typed daemon decisions. GTK dialogs contain no trust logic,
// and privileged requests go back through the authenticated broker socket.
func runUSBReview(args []string, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "FAIL: review takes no arguments")
		return 2
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()
	executable, err := os.Executable()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	seen := map[string]bool{}
	for ctx.Err() == nil {
		requestCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		response, err := broker.Call(requestCtx, broker.DefaultSocket, broker.Request{Action: broker.ActionPolicy})
		cancel()
		if err != nil {
			fmt.Fprintf(stderr, "WARN: USB review: %v\n", err)
		} else {
			active := map[string]bool{}
			for _, d := range response.Policy {
				active[d.Connection] = true
				if seen[d.Connection] || d.Target != usbtrust.TargetBlock {
					continue
				}
				seen[d.Connection] = true
				text := fmt.Sprintf("%s (%s)\nSerial: %s\nPort: %s\nParent: %s\nInterfaces: %s\nReason: %s\n", d.Identity.Name, d.Identity.VIDPID, d.Identity.Serial, d.Identity.Port, d.Identity.ParentHash, strings.Join(d.Identity.Interfaces, ", "), d.Reason)
				if d.Role != "" {
					text += "ODDC role: " + d.Role + "\n"
				}
				if d.TrustedID != "" {
					text += "Trusted record: " + d.TrustedID + "\n"
				}
				text += "\nUSB descriptors and serials can be spoofed.\n"
				for _, risk := range d.Risks {
					// Risk text is presentation only; decisions remain daemon-owned.
					text += "\n" + string(risk.Severity) + ": " + risk.Detail
				}

				if !response.Enforcing {
					text = "AUDIT MODE: USB blocking is not active. Keep blocked leaves the derived blocked decision unchanged; Allow once is memory-only for this connection; signed trust and enrollment choices persist but are not enforced until enforcement is enabled.\n\n" + text
				}

				dialog := []string{"--list", "--radiolist", "--title=USB device review", "--text=" + html.EscapeString(wrapUSBReviewText(text, 72)), "--column=Choose", "--column=Action", "--column=Meaning", "--hide-column=2", "--print-column=2"}
				rows, available := usbReviewChoices(d)
				if len(rows) == 0 {
					continue
				}
				// Announce the block quietly; the review dialog opens only on
				// request. Dismissing the notice keeps the device blocked.
				notice := exec.CommandContext(ctx, "notify-send", usbReviewNotice(d, response.Enforcing)...)
				picked, err := notice.Output()
				if err != nil || !usbReviewRequested(string(picked)) {
					continue
				}
				dialog = append(dialog, rows...)
				choice, err := exec.CommandContext(ctx, "zenity", dialog...).Output()
				if err != nil {
					continue
				}
				action := strings.TrimSpace(string(choice))
				if !available[action] {
					continue
				}
				// Rejecting a device never requires authentication. The derived
				// policy is already block; keeping it blocked only dismisses this
				// review for the current connection.
				if action == string(usbtrust.ActionKeepBlocked) {
					continue
				}
				request := []string{"usb", action, "--runtime-id", d.RuntimeID, "--connection", d.Connection}
				switch action {
				case "accept-replacement":
					request = append(request, "--trusted-id", d.TrustedID)
				case "enroll-internal":
					request = append(request, "--role", d.Role)
				case "trust-permanent":
					portability, err := exec.CommandContext(ctx, "zenity", "--list", "--radiolist", "--title=Permanent USB trust", "--text=Choose whether this device may move between ports and docks on this machine.", "--column=Choose", "--column=Portable", "--column=Meaning", "--hide-column=2", "--print-column=2", "TRUE", "false", "Bind to this topology", "FALSE", "true", "Allow on other ports and docks").Output()
					if err != nil {
						continue
					}
					value := strings.TrimSpace(string(portability))
					if value != "true" && value != "false" {
						continue
					}
					request = append(request, "--portable="+value)
				}
				if output, err := usbAuthorize(ctx, d.Identity.Name, append([]string{executable}, request...)); err != nil {
					_, _ = exec.CommandContext(ctx, "zenity", "--error", "--title=USB decision failed", "--text="+html.EscapeString(string(output))).Output()
					delete(seen, d.Connection)
				}
			}
			for token := range seen {
				if !active[token] {
					delete(seen, token)
				}
			}
		}
		select {
		case <-ctx.Done():
		case <-time.After(2 * time.Second):
		}
	}
	return 0
}

// usbReviewNotice builds the notify-send arguments announcing a device that
// needs a decision. In audit mode the device already works, so the notice
// must not claim it is blocked.
func usbReviewNotice(d usbtrust.Decision, enforcing bool) []string {
	name := d.Identity.Name
	if name == "" {
		name = "Unknown device"
	}
	summary, dismiss := "USB device blocked", "--action=keep=Keep blocked"
	if !enforcing {
		summary, dismiss = "Untrusted USB device (audit mode, not blocked)", "--action=keep=Ignore"
	}
	return []string{
		"--app-name=USB Guard",
		"--icon=drive-removable-media-usb",
		"--expire-time=0",
		"--action=default=Review",
		"--action=review=Review…",
		dismiss,
		summary,
		// Notification bodies may carry markup; device strings are untrusted.
		html.EscapeString(fmt.Sprintf("%s (%s)", name, d.Identity.VIDPID)),
	}
}

// wrapUSBReviewText breaks long lines at spaces. Zenity never wraps the list
// dialog text, so one long line would set a minimum width wider than the
// screen and override the compositor's size rule.
func wrapUSBReviewText(text string, width int) string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		current := ""
		for _, word := range strings.Fields(line) {
			if current != "" && len([]rune(current))+1+len([]rune(word)) > width {
				out = append(out, current)
				current = ""
			}
			if current != "" {
				current += " "
			}
			current += word
		}
		out = append(out, current)
	}
	return strings.Join(out, "\n")
}

// usbAuthorize runs one USB trust mutation behind fresh authentication. The
// fingerprint is tried first while a notice offers the password instead; a
// failed or skipped scan opens the password prompt at once. The credential
// cached for the command is dropped again afterwards.
func usbAuthorize(ctx context.Context, name string, command []string) ([]byte, error) {
	const sudo = "/run/wrappers/bin/sudo"
	authHelper, err := siblingExecutable("gjallar-sudo-auth")
	if err != nil {
		return []byte(err.Error()), err
	}
	_ = exec.CommandContext(ctx, sudo, "-k").Run()
	defer func() { _ = exec.Command(sudo, "-k").Run() }()

	if !usbFingerprintAuth(ctx, sudo, authHelper, name) {
		// pam_askpass_service limits this phase to the password.
		if output, err := exec.CommandContext(ctx, sudo, "-A", authHelper).CombinedOutput(); err != nil {
			return append([]byte("Authentication failed.\n"), output...), err
		}
	}
	return exec.CommandContext(ctx, sudo, append([]string{"-n", "--"}, command...)...).CombinedOutput()
}

// usbFingerprintAuth waits for the fingerprint-only sudo phase while a notice
// explains what is happening. Choosing "Use password" cancels the scan.
func usbFingerprintAuth(ctx context.Context, sudo, authHelper, name string) bool {
	scanCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	scan := exec.CommandContext(scanCtx, sudo, authHelper)
	scan.Cancel = func() error { return scan.Process.Signal(syscall.SIGTERM) }
	if err := scan.Start(); err != nil {
		return false
	}
	scanned := make(chan error, 1)
	go func() { scanned <- scan.Wait() }()

	// SIGINT makes notify-send close the notice it is waiting on.
	notice := exec.CommandContext(scanCtx, "notify-send", usbFingerprintNotice(name)...)
	notice.Cancel = func() error { return notice.Process.Signal(os.Interrupt) }
	picked := make(chan string, 1)
	go func() {
		output, _ := notice.Output()
		picked <- string(output)
	}()

	select {
	case err := <-scanned:
		return err == nil
	case output := <-picked:
		if strings.TrimSpace(output) == "password" {
			cancel()
			<-scanned
			return false
		}
		// A dismissed notice leaves the scan running.
		return <-scanned == nil
	}
}

// usbFingerprintNotice builds the notify-send arguments for the scan prompt.
func usbFingerprintNotice(name string) []string {
	if name == "" {
		name = "this USB device"
	}
	return []string{
		"--app-name=USB Guard",
		"--icon=fingerprint",
		"--expire-time=0",
		"--action=password=Use password",
		"Touch the fingerprint reader",
		html.EscapeString("to authorize " + name),
	}
}

// usbReviewRequested reports whether the notice was answered with Review or a
// click on its body.
func usbReviewRequested(output string) bool {
	switch strings.TrimSpace(output) {
	case "review", "default":
		return true
	}
	return false
}

// The domain supplies available actions; this adapter only gives them labels.
func usbReviewChoices(d usbtrust.Decision) ([]string, map[string]bool) {
	var rows []string
	available := map[string]bool{}
	for _, action := range d.Actions {
		label, selected := "", "FALSE"
		switch action {
		case usbtrust.ActionKeepBlocked:
			label, selected = "Keep blocked", "TRUE"
		case usbtrust.ActionAllowOnce:
			label = "Allow once, until disconnect"
		case usbtrust.ActionTrustPermanent:
			label = "Trust permanently on this machine"
		case usbtrust.ActionEnrollInternal:
			label = "Enroll as expected internal role " + d.Role
		case usbtrust.ActionAcceptReplacement:
			label = "Accept replacement for " + d.TrustedID
		default:
			continue
		}
		rows = append(rows, selected, string(action), label)
		available[string(action)] = true
	}
	return rows, available
}
