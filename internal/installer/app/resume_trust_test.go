package app

import (
	"os/exec"
	"strings"
	"testing"
)

func TestResumeTrustsOnlyTheTransactionRepository(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}

	t.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "user.name")
	t.Setenv("GIT_CONFIG_VALUE_0", "kept")

	if err := trustResumeRepository("/home/tester/Documents/gjallarOS"); err != nil {
		t.Fatal(err)
	}

	for key, want := range map[string]string{
		"safe.directory": "/home/tester/Documents/gjallarOS",
		"user.name":      "kept",
	} {
		cmd := exec.Command("git", "config", "--get-all", key)
		cmd.Dir = t.TempDir()
		out, err := cmd.Output()
		if err != nil || strings.TrimSpace(string(out)) != want {
			t.Fatalf("git config %s = %q, %v; want %q", key, out, err, want)
		}
	}

	t.Setenv("GIT_CONFIG_COUNT", "x")
	if trustResumeRepository("/r") == nil {
		t.Fatal("accepted a malformed GIT_CONFIG_COUNT")
	}
}
