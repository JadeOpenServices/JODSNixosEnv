package repojson

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

func CanonicalizeChangedTracked(
	ctx context.Context,
	repo string,
) ([]string, error) {
	paths := map[string]struct{}{}

	for _, args := range [][]string{
		{"diff", "--name-only", "-z", "--diff-filter=AM", "--", "*.json"},
		{"diff", "--cached", "--name-only", "-z", "--diff-filter=AM", "--", "*.json"},
	} {
		found, err := gitPaths(ctx, repo, args...)
		if err != nil {
			return nil, err
		}
		for _, path := range found {
			paths[path] = struct{}{}
		}
	}

	ordered := make([]string, 0, len(paths))
	for path := range paths {
		ordered = append(ordered, path)
	}
	sort.Strings(ordered)

	for _, relative := range ordered {
		if err := canonicalize(filepath.Join(repo, filepath.FromSlash(relative))); err != nil {
			return nil, fmt.Errorf("%s: %w", relative, err)
		}
	}

	return ordered, nil
}

func gitPaths(
	ctx context.Context,
	repo string,
	args ...string,
) ([]string, error) {
	argv := append([]string{"-C", repo}, args...)
	out, err := exec.CommandContext(ctx, "git", argv...).Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}

	var paths []string
	for _, raw := range bytes.Split(out, []byte{0}) {
		if len(raw) != 0 {
			paths = append(paths, string(raw))
		}
	}
	return paths, nil
}

func canonicalize(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read JSON: %w", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	var value any
	if err := decoder.Decode(&value); err != nil {
		return fmt.Errorf("parse JSON: %w", err)
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("parse JSON: trailing JSON value")
		}
		return fmt.Errorf("parse JSON trailing data: %w", err)
	}

	var formatted bytes.Buffer
	if err := json.Indent(
		&formatted,
		bytes.TrimSpace(data),
		"",
		"  ",
	); err != nil {
		return fmt.Errorf("format JSON: %w", err)
	}
	formatted.WriteByte('\n')

	if bytes.Equal(data, formatted.Bytes()) {
		return nil
	}

	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("stat JSON: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".gjallar-json-*")
	if err != nil {
		return fmt.Errorf("create temporary JSON: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if err := tmp.Chmod(info.Mode().Perm()); err != nil {
		tmp.Close()
		return fmt.Errorf("preserve JSON mode: %w", err)
	}
	if _, err := tmp.Write(formatted.Bytes()); err != nil {
		tmp.Close()
		return fmt.Errorf("write temporary JSON: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync temporary JSON: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary JSON: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace JSON: %w", err)
	}

	return nil
}
