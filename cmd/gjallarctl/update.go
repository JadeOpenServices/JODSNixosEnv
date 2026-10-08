package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/JadeOpenServices/gjallarOS/internal/installercheck"
)

// Standalone updates move the checkout to the newest signed release of the
// configured channel (fast-forward only) and rebuild with the repository's
// own flake.lock, so every machine runs the input versions that were tested.
// The trust root (remote, allowed signers) comes from the running system,
// never from the checkout being updated.

// updateConfigPath is written by system/tools/commands.
var updateConfigPath = "/etc/gjallar/update.json"

// updateRebuild applies the updated checkout; tests replace it.
var updateRebuild = func(stdout, stderr io.Writer) int {
	return runCommand(context.Background(), stdout, stderr, "rebuild")
}

type updateConfig struct {
	Channel        string `json:"channel"`
	Remote         string `json:"remote"`
	Release        string `json:"release"`
	AllowedSigners string `json:"allowedSigners"`
	SSHKeygen      string `json:"sshKeygen"`
}

const updateUsage = "Usage: update [--check|--inputs]"

func runUpdate(args []string, stdout, stderr io.Writer) int {
	mode := ""
	if len(args) > 1 {
		fmt.Fprintln(stderr, updateUsage)
		return 2
	}
	if len(args) == 1 {
		mode = args[0]
	}
	if mode != "" && mode != "--check" && mode != "--inputs" {
		fmt.Fprintln(stderr, updateUsage)
		return 2
	}

	repo := os.Getenv("GJALLAROS_REPO")
	if repo == "" {
		found, err := installercheck.DiscoverRepository("")
		if err != nil {
			fmt.Fprintf(stderr, "ERROR: %v\n", err)
			return 1
		}
		repo = found
	}

	if mode == "--inputs" {
		fmt.Fprintln(stderr, "WARNING: moving every flake input to its newest upstream version; GjallarOS releases were not tested with these versions.")
		return runCommand(context.Background(), stdout, stderr, "nix", "flake", "update", "--flake", repo)
	}

	cfg, err := loadUpdateConfig(updateConfigPath)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}
	target, err := fetchVerifiedRelease(repo, cfg)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}

	head, err := gitOutput(repo, nil, "rev-parse", "HEAD")
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}
	if head == target.commit {
		fmt.Fprintf(stdout, "GjallarOS is up to date (%s).\n", target.name)
		return 0
	}
	if _, err := gitOutput(repo, nil, "merge-base", "--is-ancestor", head, target.commit); err != nil {
		fmt.Fprintf(stderr, "ERROR: the checkout in %s has commits that are not in %s; not updating.\n", repo, target.name)
		return 1
	}

	changes, _ := gitOutput(repo, nil, "log", "--oneline", "--no-decorate", "-n", "20", head+".."+target.commit)
	fmt.Fprintf(stdout, "Update available: %s\n%s\n", target.name, changes)
	if mode == "--check" {
		return 0
	}

	dirty, err := gitOutput(repo, nil, "status", "--porcelain", "--untracked-files=no")
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}
	// rebuild --hardware-update and update --inputs only move flake.lock;
	// the release brings its own tested lock.
	if dirty == "M flake.lock" {
		if _, err := gitOutput(repo, nil, "checkout", "--", "flake.lock"); err != nil {
			fmt.Fprintf(stderr, "ERROR: %v\n", err)
			return 1
		}
		fmt.Fprintf(stdout, "Local flake.lock changes replaced by the lock of %s; run rebuild --hardware-update afterwards to move ODDC again.\n", target.name)
	} else if dirty != "" {
		fmt.Fprintf(stderr, "ERROR: the checkout in %s has uncommitted changes; commit or discard them first.\n", repo)
		return 1
	}
	move := []string{"merge", "--ff-only", "--quiet", target.commit}
	if _, err := gitOutput(repo, nil, "symbolic-ref", "-q", "HEAD"); err != nil {
		move = []string{"checkout", "--quiet", "--detach", target.commit}
	}
	if _, err := gitOutput(repo, nil, move...); err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "Checkout moved to %s; rebuilding.\n", target.name)
	if status := updateRebuild(stdout, stderr); status != 0 {
		fmt.Fprintln(stderr, "The checkout is updated but the system is not; run rebuild again.")
		return status
	}
	return 0
}

func loadUpdateConfig(path string) (updateConfig, error) {
	var cfg updateConfig
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("read update settings: %w", err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse %s: %w", path, err)
	}
	if cfg.Channel != "stable" && cfg.Channel != "main" {
		return cfg, fmt.Errorf("%s: unknown update channel %q", path, cfg.Channel)
	}
	if cfg.Remote == "" || cfg.Release == "" || cfg.SSHKeygen == "" {
		return cfg, fmt.Errorf("%s: remote, release and sshKeygen are required", path)
	}
	signers, err := os.ReadFile(cfg.AllowedSigners)
	if err != nil {
		return cfg, fmt.Errorf("read release signing keys: %w", err)
	}
	if !hasSigner(signers) {
		return cfg, fmt.Errorf("no release signing keys in %s; updates cannot be verified", cfg.AllowedSigners)
	}
	return cfg, nil
}

func hasSigner(data []byte) bool {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			return true
		}
	}
	return false
}

type updateTarget struct {
	name   string
	commit string
}

// Fetched refs land in their own namespace so a remote cannot overwrite the
// user's branches or tags; only a verified ref is ever checked out.
const updateRefs = "refs/gjallar-update/"

func fetchVerifiedRelease(repo string, cfg updateConfig) (updateTarget, error) {
	verify := []string{
		"-c", "gpg.format=ssh",
		"-c", "gpg.ssh.allowedSignersFile=" + cfg.AllowedSigners,
		"-c", "gpg.ssh.program=" + cfg.SSHKeygen,
	}
	if cfg.Channel == "main" {
		ref := updateRefs + "main"
		if _, err := gitOutput(repo, nil, "fetch", "--quiet", "--no-tags", cfg.Remote, "+refs/heads/main:"+ref); err != nil {
			return updateTarget{}, fmt.Errorf("fetch %s: %w", cfg.Remote, err)
		}
		if _, err := gitOutput(repo, verify, "verify-commit", ref); err != nil {
			return updateTarget{}, fmt.Errorf("main at %s is not signed by a GjallarOS release key; not updating", cfg.Remote)
		}
		commit, err := gitOutput(repo, nil, "rev-parse", ref+"^{commit}")
		return updateTarget{name: "main " + shortCommit(commit), commit: commit}, err
	}

	pattern := "v" + cfg.Release + ".*"
	refspec := fmt.Sprintf("+refs/tags/%s:%stags/%s", pattern, updateRefs, pattern)
	if _, err := gitOutput(repo, nil, "fetch", "--quiet", "--no-tags", "--prune", cfg.Remote, refspec); err != nil {
		return updateTarget{}, fmt.Errorf("fetch %s: %w", cfg.Remote, err)
	}
	list, err := gitOutput(repo, nil, "for-each-ref", "--sort=-v:refname", "--format=%(refname)", updateRefs+"tags/")
	if err != nil {
		return updateTarget{}, err
	}
	if list == "" {
		return updateTarget{}, fmt.Errorf("no GjallarOS %s release found at %s", cfg.Release, cfg.Remote)
	}
	ref := strings.SplitN(list, "\n", 2)[0]
	name := strings.TrimPrefix(ref, updateRefs+"tags/")
	if _, err := gitOutput(repo, verify, "verify-tag", ref); err != nil {
		return updateTarget{}, fmt.Errorf("release %s is not signed by a GjallarOS release key; not updating", name)
	}
	// A validly signed tag object republished under a newer name would
	// otherwise pass as that release.
	object, err := gitOutput(repo, nil, "cat-file", "tag", ref)
	if err != nil || !strings.Contains(object+"\n", "\ntag "+name+"\n") {
		return updateTarget{}, fmt.Errorf("release %s was signed under a different name; not updating", name)
	}
	commit, err := gitOutput(repo, nil, "rev-parse", ref+"^{commit}")
	return updateTarget{name: name, commit: commit}, err
}

func shortCommit(commit string) string {
	if len(commit) > 12 {
		return commit[:12]
	}
	return commit
}

func gitOutput(repo string, config []string, args ...string) (string, error) {
	full := append([]string{"-C", repo}, config...)
	cmd := exec.Command("git", append(full, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return "", fmt.Errorf("git %s: %s", args[0], msg)
		}
		return "", fmt.Errorf("git %s: %w", args[0], err)
	}
	return strings.TrimSpace(string(out)), nil
}
