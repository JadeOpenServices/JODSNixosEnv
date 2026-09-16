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

var machineLocalPaths = []string{
	"user.config.json",
	"generated/state.nix",
	"generated/hardware.nix",
}

func trackedPath(ctx context.Context, root, relative string) bool {
	tracked, _ := git(
		ctx,
		root,
		"ls-files",
		"--error-unmatch",
		"--",
		relative,
	)
	return len(tracked) != 0
}

func excludePatterns() []string {
	return []string{
		"user.config.json",
		".user.config.json-*",
		"generated/",
		".vm/",
		"non-nix/wallpapers/user-*",
	}
}

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

	protected := []string{}

	for _, pattern := range machineLocalPaths {
		matches, err := filepath.Glob(filepath.Join(root, pattern))
		if err != nil {
			return protected, fmt.Errorf(
				"expand machine-local pattern %q: %w",
				pattern,
				err,
			)
		}

		for _, path := range matches {
			relative, err := filepath.Rel(root, path)
			if err != nil {
				return protected, err
			}

			if trackedPath(ctx, root, relative) {
				return protected, fmt.Errorf(
					"machine-local state must never be tracked by Git: %s",
					relative,
				)
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
			if trackedPath(ctx, root, relative) {
				return protected, fmt.Errorf(
					"machine-local hardware state must never be tracked by Git: %s",
					relative,
				)
			}
		} else if !os.IsNotExist(err) {
			return protected, err
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

	contents, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read Git exclude: %w", err)
	}

	const begin = "# BEGIN GjallarOS machine-local state"
	const end = "# END GjallarOS machine-local state"

	legacy := map[string]bool{
		"profiles/*/hardware-configuration.nix.bak.*": true,
		"profiles/*/hardware-configuration.nix":       true,
		"settings.nix":                                true,
		".settings.nix-*":                             true,
		"system/hardware/generated.nix":               true,
		"system/hardware/.generated.nix-*":            true,
		"system/hardware/generated.nix.bak.*":         true,
		"user.config.json":                            true,
		".user.config.json-*":                         true,
		"generated/":                                  true,
	}

	var preserved []string
	inManagedBlock := false

	scanner := bufio.NewScanner(strings.NewReader(string(contents)))
	for scanner.Scan() {
		line := scanner.Text()

		if line == begin {
			inManagedBlock = true
			continue
		}
		if line == end {
			inManagedBlock = false
			continue
		}
		if inManagedBlock || legacy[line] {
			continue
		}

		preserved = append(preserved, line)
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	for len(preserved) > 0 && preserved[len(preserved)-1] == "" {
		preserved = preserved[:len(preserved)-1]
	}

	preserved = append(preserved, "", begin)
	preserved = append(preserved, excludePatterns()...)
	preserved = append(preserved, end, "")

	updated := []byte(strings.Join(preserved, "\n"))

	tmp, err := os.CreateTemp(filepath.Dir(path), ".exclude-*")
	if err != nil {
		return fmt.Errorf("create Git exclude: %w", err)
	}
	name := tmp.Name()
	defer os.Remove(name)

	if err := tmp.Chmod(0644); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(updated); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return fmt.Errorf("replace Git exclude: %w", err)
	}

	return nil
}

func git(ctx context.Context, root string, args ...string) ([]byte, error) {
	argv := append([]string{"-C", root}, args...)
	out, err := exec.CommandContext(ctx, "git", argv...).Output()
	if err != nil {
		return out, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return out, nil
}
