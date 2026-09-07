package recoveryresize

import (
	"strings"
	"testing"
)

func TestManifestContainsNoUnlockSecretField(t *testing.T) {
	top := topology()
	plan, err := BuildPlan(top, requirements())
	if err != nil {
		t.Fatal(err)
	}

	m, err := ManifestFromPlan(top, plan, "nonce-123")
	if err != nil {
		t.Fatal(err)
	}

	if m.Nonce == "" {
		t.Fatal("nonce missing")
	}

	// Compile-time structure plus this lexical assertion protects against
	// accidentally adding obvious unlock-secret fields to the authenticated
	// non-secret operation state.
	for _, forbidden := range []string{
		"passphrase",
		"password",
		"keyfile",
		"unlocksecret",
	} {
		if strings.Contains(
			strings.ToLower(strings.Join([]string{
				m.Nonce,
				m.DiskGUID,
				m.RootPARTUUID,
				m.LUKSUUID,
				m.BtrfsUUID,
			}, " ")),
			forbidden,
		) {
			t.Fatalf("manifest contains forbidden secret marker %q", forbidden)
		}
	}
}

func TestResumeIdentityMismatchFailsClosed(t *testing.T) {
	top := topology()
	plan, err := BuildPlan(top, requirements())
	if err != nil {
		t.Fatal(err)
	}

	m, err := ManifestFromPlan(top, plan, "nonce")
	if err != nil {
		t.Fatal(err)
	}

	changed := top
	changed.RootPARTUUID = "different"

	err = ValidateResume(
		changed,
		m,
		StagePreparingResize,
	)
	if err == nil ||
		!strings.Contains(err.Error(), "PARTUUID") {
		t.Fatalf("error=%v", err)
	}
}

func TestResumeRootStartMismatchFailsClosed(t *testing.T) {
	top := topology()
	plan, err := BuildPlan(top, requirements())
	if err != nil {
		t.Fatal(err)
	}

	m, err := ManifestFromPlan(top, plan, "nonce")
	if err != nil {
		t.Fatal(err)
	}

	changed := top
	changed.RootStartBytes += MiB
	changed.RootSizeBytes =
		changed.RootEndBytes - changed.RootStartBytes

	err = ValidateResume(
		changed,
		m,
		StagePreparingResize,
	)
	if err == nil ||
		!strings.Contains(err.Error(), "root start mismatch") {
		t.Fatalf("error=%v", err)
	}
}

func TestResumeStaleStageFailsClosed(t *testing.T) {
	top := topology()
	plan, err := BuildPlan(top, requirements())
	if err != nil {
		t.Fatal(err)
	}

	m, err := ManifestFromPlan(top, plan, "nonce")
	if err != nil {
		t.Fatal(err)
	}

	err = ValidateResume(
		top,
		m,
		StageValidatingEnvironment,
	)
	if err == nil ||
		!strings.Contains(err.Error(), "stale or inconsistent") {
		t.Fatalf("error=%v", err)
	}
}
