package main

import (
	"reflect"
	"testing"

	"github.com/bakanura/gjallarOS/internal/usbtrust"
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
				"FALSE", "allow-once", "Allow once, until disconnect",
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
	args := usbReviewNotice(d)
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
