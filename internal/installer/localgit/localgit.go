package localgit

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var excludePatterns = []string{"profiles/*/hardware-configuration.nix.bak.*", "user.config.json", "non-nix/wallpapers/user-*"}

func Protect(ctx context.Context, repo, hardware string) ([]string, error) {
	root, err := filepath.Abs(repo)
	if err != nil {
		return nil, fmt.Errorf("resolve repository: %w", err)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return nil, fmt.Errorf("resolve repository: %w", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".git")); err != nil {
		return nil, fmt.Errorf("Git metadata not found: %w", err)
	}
	gitDirBytes, err := git(ctx, root, "rev-parse", "--git-dir")
	if err != nil {
		return nil, err
	}
	gitDir := strings.TrimSpace(string(gitDirBytes))
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(root, gitDir)
	}
	if err := updateExclude(filepath.Join(gitDir, "info", "exclude")); err != nil {
		return nil, err
	}
	var protected []string
	for _, relative := range []string{"settings.nix", "user.config.json"} {
		if _, err := os.Stat(filepath.Join(root, relative)); err != nil {
			continue
		}
		tracked, _ := git(ctx, root, "ls-files", "--error-unmatch", "--", relative)
		if relative == "settings.nix" || len(tracked) != 0 {
			if _, err := git(ctx, root, "update-index", "--skip-worktree", "--", relative); err != nil {
				return protected, err
			}
			protected = append(protected, relative)
		}
	}
	if hardware != "" {
		relative, err := containedRelative(root, hardware)
		if err != nil {
			return protected, err
		}
		if _, err := os.Stat(filepath.Join(root, relative)); err == nil {
			if _, err := git(ctx, root, "update-index", "--skip-worktree", "--", relative); err != nil {
				return protected, err
			}
			protected = append(protected, relative)
		}
	}
	matches, _ := filepath.Glob(filepath.Join(root, "profiles", "*", "details.nix"))
	for _, path := range matches {
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return protected, err
		}
		if out, _ := git(ctx, root, "ls-files", "--error-unmatch", "--", relative); len(out) == 0 {
			if _, err := git(ctx, root, "add", "--", relative); err != nil {
				return protected, err
			}
		}
	}
	return protected, nil
}

func containedRelative(root, path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("target is outside repository: %s", path)
	}
	return rel, nil
}

func updateExclude(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create Git info directory: %w", err)
	}
	existing := map[string]bool{}
	if f, err := os.Open(path); err == nil {
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			existing[scanner.Text()] = true
		}
		f.Close()
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open Git exclude: %w", err)
	}
	defer f.Close()
	for _, pattern := range excludePatterns {
		if !existing[pattern] {
			if _, err := fmt.Fprintln(f, pattern); err != nil {
				return fmt.Errorf("write Git exclude: %w", err)
			}
		}
	}
	return f.Sync()
}

func git(ctx context.Context, root string, args ...string) ([]byte, error) {
	argv := append([]string{"-C", root}, args...)
	out, err := exec.CommandContext(ctx, "git", argv...).Output()
	if err != nil {
		return out, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return out, nil
}
