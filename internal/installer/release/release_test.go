package release

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInspect(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "deployment"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "deployment", "release-policy.json"), []byte(`{"release":"26.05"}`), 0644); err != nil {
		t.Fatal(err)
	}
	osr := filepath.Join(root, "os-release")
	if err := os.WriteFile(osr, []byte("ID=nixos\nVERSION_ID=\"26.05\"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	expected, actual, err := Inspect(root, osr)
	if err != nil || expected != "26.05" || actual != "26.05" {
		t.Fatalf("%q %q %v", expected, actual, err)
	}
}

func TestExpected(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "deployment"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(root, "deployment", "release-policy.json"),
		[]byte(`{"release":"26.05"}`),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	expected, err := Expected(root)
	if err != nil {
		t.Fatal(err)
	}
	if expected != "26.05" {
		t.Fatalf("expected=%q", expected)
	}
}

func TestExpectedRejectsMissingRelease(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "deployment"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(root, "deployment", "release-policy.json"),
		[]byte(`{"release":""}`),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	if _, err := Expected(root); err == nil {
		t.Fatal("empty pinned release accepted")
	}
}
