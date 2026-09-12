package oddcvalidation

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestContributorAuthorityAllowsVerifiedWriter(t *testing.T) {
	run := func(
		ctx context.Context,
		dir string,
		name string,
		args ...string,
	) ([]byte, error) {
		cmd := strings.Join(append([]string{name}, args...), " ")

		switch {
		case cmd == "git remote get-url origin":
			return []byte("https://github.com/bakanura/JODSNixosEnv.git\n"), nil

		case cmd == "git status --porcelain --untracked-files=all":
			return nil, nil

		case cmd == "git symbolic-ref --quiet --short HEAD":
			return []byte("feature/gjal-83-track-validated-device-releases\n"), nil

		case cmd == "gh api user":
			return []byte(`{"login":"bakanura"}`), nil

		case cmd == "gh api repos/bakanura/JODSNixosEnv":
			return []byte(`{
			  "full_name":"bakanura/JODSNixosEnv",
			  "permissions":{"push":true}
			}`), nil

		default:
			return nil, errors.New("unexpected command: " + cmd)
		}
	}

	got := checkContributorAuthority(
		context.Background(),
		"/repo",
		"bakanura/JODSNixosEnv",
		run,
	)

	if !got.Allowed {
		t.Fatalf("verified contributor rejected: %+v", got)
	}
	if got.Identity != "bakanura" {
		t.Fatalf("Identity=%q", got.Identity)
	}
	if got.Repo != "bakanura/JODSNixosEnv" {
		t.Fatalf("Repo=%q", got.Repo)
	}
}

func TestContributorAuthorityFallsBackForDirtyCheckout(t *testing.T) {
	ghCalled := false

	run := func(
		ctx context.Context,
		dir string,
		name string,
		args ...string,
	) ([]byte, error) {
		cmd := strings.Join(append([]string{name}, args...), " ")

		switch {
		case cmd == "git remote get-url origin":
			return []byte("https://github.com/bakanura/JODSNixosEnv.git\n"), nil

		case cmd == "git status --porcelain --untracked-files=all":
			return []byte(" M internal/installer/app/app.go\n"), nil

		case strings.HasPrefix(cmd, "gh "):
			ghCalled = true
			return nil, nil

		default:
			return nil, errors.New("unexpected command: " + cmd)
		}
	}

	got := checkContributorAuthority(
		context.Background(),
		"/repo",
		"bakanura/JODSNixosEnv",
		run,
	)

	if got.Allowed {
		t.Fatalf("dirty checkout was authorized: %+v", got)
	}
	if got.Reason != "checkout contains unrelated changes" {
		t.Fatalf("Reason=%q", got.Reason)
	}
	if ghCalled {
		t.Fatal("GitHub authority was queried after dirty checkout already denied writeback")
	}
}

func TestNormalizeGitHubRepo(t *testing.T) {
	tests := map[string]string{
		"https://github.com/bakanura/JODSNixosEnv.git": "bakanura/JODSNixosEnv",
		"git@github.com:bakanura/JODSNixosEnv.git":     "bakanura/JODSNixosEnv",
	}

	for input, want := range tests {
		got, err := normalizeGitHubRepo(input)
		if err != nil {
			t.Fatalf("%q: %v", input, err)
		}
		if got != want {
			t.Fatalf("%q -> %q want %q", input, got, want)
		}
	}
}
