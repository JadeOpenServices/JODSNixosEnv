package main

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// fingerprintEnrolled reports whether the calling user has a print enrolled
// on the default fprintd reader. The fingerprint-only PAM service fails at
// once without one, and sudo counts each failure as a wrong password: three
// "Sorry, try again" before the password prompt (real HP, 2026-10-09). Any
// error means no usable fingerprint, so authentication goes straight to the
// password.
func fingerprintEnrolled(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	device, err := exec.CommandContext(
		ctx,
		"busctl", "--system", "call",
		"net.reactivated.Fprint",
		"/net/reactivated/Fprint/Manager",
		"net.reactivated.Fprint.Manager",
		"GetDefaultDevice",
	).Output()
	if err != nil {
		return false
	}
	path, ok := parseBusctlObjectPath(string(device))
	if !ok {
		return false
	}

	// An empty user name means the caller; fprintd answers with an error
	// when that user has no prints.
	fingers, err := exec.CommandContext(
		ctx,
		"busctl", "--system", "call",
		"net.reactivated.Fprint",
		path,
		"net.reactivated.Fprint.Device",
		"ListEnrolledFingers",
		"s", "",
	).Output()
	if err != nil {
		return false
	}
	return busctlArrayLength(string(fingers)) > 0
}

func parseBusctlObjectPath(value string) (string, bool) {
	fields := strings.Fields(value)
	if len(fields) != 2 || fields[0] != "o" {
		return "", false
	}
	path := strings.Trim(fields[1], `"`)
	if !strings.HasPrefix(path, "/net/reactivated/Fprint/Device/") {
		return "", false
	}
	return path, true
}

func busctlArrayLength(value string) int {
	fields := strings.Fields(value)
	if len(fields) < 2 || fields[0] != "as" {
		return 0
	}
	n, err := strconv.Atoi(fields[1])
	if err != nil || n < 0 {
		return 0
	}
	return n
}
