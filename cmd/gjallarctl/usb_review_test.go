package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/JadeOpenServices/gjallarOS/internal/usbtrust"
)

func TestUSBReviewUsesOnlyDomainActions(t *testing.T) {
	for _, tc := range []struct {
		name     string
		decision usbtrust.Decision
	}{
		{
			name: "ambiguous internal expectation",
			decision: usbtrust.Decision{
				Reason: string(usbtrust.CodeInternalAmbiguous),
				Role:   "camera",
			},
		},
		{
			name: "unexpected hardwired device",
			decision: usbtrust.Decision{
				ObservedDevice: usbtrust.ObservedDevice{
					Identity: usbtrust.Identity{ConnectType: "hardwired"},
				},
			},
		},
		{
			name: "unchanged accepted internal device kept blocked",
			decision: usbtrust.Decision{
				Reason:    "blocked-for-connection",
				Role:      "camera",
				TrustedID: "device:camera",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.decision.Actions = []usbtrust.Action{usbtrust.ActionKeepBlocked, usbtrust.ActionAllowOnce}
			rows, available := usbReviewChoices(tc.decision)
			wantRows := []string{
				"TRUE", "keep-blocked", "Keep blocked",
				"FALSE", "allow-once", "Allow until unplugged",
			}
			if !reflect.DeepEqual(rows, wantRows) {
				t.Fatalf("review invented or mislabeled an action: got %q, want %q", rows, wantRows)
			}
			wantAvailable := map[string]bool{"keep-blocked": true, "allow-once": true}
			if !reflect.DeepEqual(available, wantAvailable) {
				t.Fatalf("review accepts actions absent from domain decision: %v", available)
			}
		})
	}
}

func TestUSBReviewLabelsReplacementExplicitly(t *testing.T) {
	d := usbtrust.Decision{
		Role:      "camera",
		TrustedID: "device:accepted-camera",
		Actions:   []usbtrust.Action{usbtrust.ActionAcceptReplacement},
	}
	rows, available := usbReviewChoices(d)
	wantRows := []string{"FALSE", "accept-replacement", "Accept replacement for device:accepted-camera"}
	if !reflect.DeepEqual(rows, wantRows) {
		t.Fatalf("replacement must identify the record being replaced: got %q, want %q", rows, wantRows)
	}
	if !reflect.DeepEqual(available, map[string]bool{"accept-replacement": true}) {
		t.Fatalf("unexpected available actions: %v", available)
	}
}

func TestUSBReviewIgnoresUnknownActions(t *testing.T) {
	for _, actions := range [][]usbtrust.Action{
		{"future-action"},
		{"future-action", usbtrust.ActionKeepBlocked, "another-future-action"},
	} {
		d := usbtrust.Decision{Actions: actions}
		rows, available := usbReviewChoices(d)
		if available["future-action"] || available["another-future-action"] {
			t.Fatalf("unknown action became requestable: %v", available)
		}
		if len(actions) == 1 {
			if len(rows) != 0 || len(available) != 0 {
				t.Fatalf("unknown action produced review choices: %q, %v", rows, available)
			}
			continue
		}
		if !reflect.DeepEqual(rows, []string{"TRUE", "keep-blocked", "Keep blocked"}) ||
			!reflect.DeepEqual(available, map[string]bool{"keep-blocked": true}) {
			t.Fatalf("unknown action altered known choices: %q, %v", rows, available)
		}
	}
}

func TestUSBReviewNoticeEscapesDeviceStrings(t *testing.T) {
	d := usbtrust.Decision{ObservedDevice: usbtrust.ObservedDevice{
		Identity: usbtrust.Identity{Name: `<b>Keyboard</b>`, VIDPID: "dead:beef"},
	}}
	args := usbReviewNotice(d, true)
	body := args[len(args)-1]
	if body != "&lt;b&gt;Keyboard&lt;/b&gt; (dead:beef)" {
		t.Fatalf("device name reached notification markup: %q", body)
	}
}

func TestUSBReviewOpensOnlyOnRequest(t *testing.T) {
	for output, want := range map[string]bool{
		"review\n":  true,
		"default\n": true,
		"keep\n":    false,
		"":          false, // dismissed or expired: stays blocked
	} {
		if got := usbReviewRequested(output); got != want {
			t.Fatalf("usbReviewRequested(%q) = %v, want %v", output, got, want)
		}
	}
}

func TestUSBReviewNoticeDoesNotClaimBlockInAuditMode(t *testing.T) {
	args := usbReviewNotice(usbtrust.Decision{}, false)
	for _, arg := range args {
		if arg == "USB device blocked" || arg == "--action=keep=Keep blocked" {
			t.Fatalf("audit-mode notice claims the device is blocked: %q", args)
		}
	}
}

func TestUSBReviewTextWrapsLongLines(t *testing.T) {
	long := strings.Repeat("word ", 40) + "\n\nshort"
	for _, line := range strings.Split(wrapUSBReviewText(long, 30), "\n") {
		if len(line) > 30 {
			t.Fatalf("line exceeds width: %q", line)
		}
	}
	if !strings.HasSuffix(wrapUSBReviewText(long, 30), "\n\nshort") {
		t.Fatal("wrapping dropped paragraph breaks")
	}
}

func TestUSBReviewSummaryFlagsAttackPatternsFirst(t *testing.T) {
	d := usbtrust.Decision{ObservedDevice: usbtrust.ObservedDevice{
		Identity: usbtrust.Identity{Name: "<i>stick</i>", VIDPID: "dead:beef", Serial: "S1", Interfaces: []string{"03:01:01", "08:06:50"}},
	}, Reason: string(usbtrust.CodeUnknownExternal)}
	var err error
	if d.Risks, err = usbtrust.AssessIdentityRisk(d.Identity); err != nil {
		t.Fatal(err)
	}
	summary := usbReviewSummary(d, true, true)
	if strings.Contains(summary, "<i>stick") {
		t.Fatalf("device name reached dialog markup: %q", summary)
	}
	bad := strings.Index(summary, usbBad+`">●</span> Stores files and can type`)
	fine := strings.Index(summary, usbFine+`">●</span> Has its own serial number`)
	if bad < 0 || fine < 0 || bad > fine {
		t.Fatalf("attack pattern missing or listed after reassurance:\n%s", summary)
	}
}

func TestUSBReviewSimpleViewStaysPlain(t *testing.T) {
	d := usbtrust.Decision{ObservedDevice: usbtrust.ObservedDevice{
		Identity: usbtrust.Identity{Name: "stick", VIDPID: "dead:beef", Interfaces: []string{"03:01:01", "08:06:50"}},
	}, Reason: string(usbtrust.CodeUnknownExternal)}
	var err error
	if d.Risks, err = usbtrust.AssessIdentityRisk(d.Identity); err != nil {
		t.Fatal(err)
	}
	summary := usbReviewSummary(d, true, false)
	if !strings.Contains(summary, usbBad+`">●</span> This device may be unsafe`) {
		t.Fatalf("simple view hides the attack pattern:\n%s", summary)
	}
	if strings.Contains(summary, "dead:beef") || strings.Contains(summary, "Can type") {
		t.Fatalf("simple view is not plain:\n%s", summary)
	}
}

func TestUSBReviewAlwaysTechnicalPreference(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv(usbReviewViewEnv, "")
	if usbReviewAlwaysTechnical() {
		t.Fatal("review must start in the simple view by default")
	}
	t.Setenv(usbReviewViewEnv, usbReviewTechnicalV)
	if !usbReviewAlwaysTechnical() {
		t.Fatal("user.config.json default was ignored")
	}
	setUSBReviewAlwaysTechnical(false)
	if usbReviewAlwaysTechnical() {
		t.Fatal("unticking the box must override the config default")
	}
	t.Setenv(usbReviewViewEnv, "")
	setUSBReviewAlwaysTechnical(true)
	if !usbReviewAlwaysTechnical() {
		t.Fatal("ticked box was not remembered")
	}
}

func TestUnpluggedConnectionIsInactive(t *testing.T) {
	policy := []usbtrust.Decision{{Connection: "a"}, {Connection: "b"}}
	if !usbConnectionActive(policy, "b") {
		t.Fatal("present connection reported inactive")
	}
	if usbConnectionActive(policy, "c") || usbConnectionActive(nil, "a") {
		t.Fatal("unplugged connection reported active")
	}
}
