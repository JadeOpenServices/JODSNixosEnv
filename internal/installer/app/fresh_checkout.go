package app

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/JadeOpenServices/gjallarOS/internal/installer/flakesource"
)

// copyFreshCheckout puts the installer's checkout at the dotfiles path of the
// freshly installed system. /etc/gjallar/repository names that path, and
// rebuild, update and the Secure Boot continuation read generated/ and
// user.config.json from it. Without the copy the checkout only lived on the
// live media and was gone after the first reboot.
func copyFreshCheckout(
	ctx context.Context,
	repo string,
	targetRoot string,
	dotfilesDir string,
	username string,
	out io.Writer,
) error {
	dir := filepath.Clean(dotfilesDir)
	if !filepath.IsAbs(dotfilesDir) || dir == "/" {
		return fmt.Errorf("dotfiles path must be an absolute directory: %q", dotfilesDir)
	}
	uid, gid, home, err := targetAccount(filepath.Join(targetRoot, "etc", "passwd"), username)
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(repo)
	if err != nil {
		return fmt.Errorf("read checkout %s: %w", repo, err)
	}
	var sources []string
	for _, entry := range entries {
		if !flakesource.Scratch(entry.Name()) {
			sources = append(sources, filepath.Join(repo, entry.Name()))
		}
	}
	if len(sources) == 0 {
		return fmt.Errorf("checkout %s is empty", repo)
	}

	owner := fmt.Sprintf("%d:%d", uid, gid)
	destination := filepath.Join(targetRoot, dir)
	parent := filepath.Dir(destination)

	// Parents inside the user's home belong to the user, like a checkout the
	// user cloned there; parents elsewhere stay root's.
	var owned []string
	if rel, err := filepath.Rel(home, filepath.Dir(dir)); err == nil &&
		rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		owned = append(owned, filepath.Join(targetRoot, home))
		if rel != "." {
			path := filepath.Join(targetRoot, home)
			for _, part := range strings.Split(rel, string(filepath.Separator)) {
				path = filepath.Join(path, part)
				owned = append(owned, path)
			}
		}
	}

	steps := [][]string{}
	if len(owned) > 0 {
		// NixOS activation already made the home 0700; -m only applies
		// when it is missing.
		steps = append(steps, []string{"mkdir", "-p", "-m", "0700", "--", owned[0]})
	}
	steps = append(steps, []string{"mkdir", "-p", "--", parent})
	if len(owned) > 0 {
		steps = append(steps, append([]string{"chown", "-h", owner, "--"}, owned...))
	}
	// No -p: never merge into a directory that is already there.
	steps = append(steps, []string{"mkdir", "-m", "0755", "--", destination})
	steps = append(steps, append(append([]string{"cp", "-a", "--"}, sources...), destination+"/"))
	steps = append(steps, []string{"chown", "-R", "-P", "-h", owner, "--", destination})

	created := false
	for _, step := range steps {
		if _, err := privilegedCommand(ctx, step...); err != nil {
			if created {
				_, _ = privilegedCommand(context.WithoutCancel(ctx), "rm", "-rf", "--", destination)
			}
			return fmt.Errorf("copy checkout to %s: %w", destination, err)
		}
		if step[0] == "mkdir" && step[len(step)-1] == destination {
			created = true
		}
	}
	fmt.Fprintf(out, "PASS: checkout copied to %s, owned by %s\n", dir, username)
	return nil
}

// targetAccount returns uid, gid and home of username from the target's
// passwd, which nixos-install's activation has just written.
func targetAccount(passwd, username string) (int, int, string, error) {
	file, err := os.Open(passwd)
	if err != nil {
		return 0, 0, "", fmt.Errorf("read target accounts: %w", err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), ":")
		if len(fields) < 7 || fields[0] != username {
			continue
		}
		uid, err := strconv.Atoi(fields[2])
		if err != nil {
			return 0, 0, "", fmt.Errorf("target account %s has uid %q", username, fields[2])
		}
		gid, err := strconv.Atoi(fields[3])
		if err != nil {
			return 0, 0, "", fmt.Errorf("target account %s has gid %q", username, fields[3])
		}
		home := filepath.Clean(fields[5])
		if uid == 0 || !filepath.IsAbs(home) || home == "/" {
			return 0, 0, "", fmt.Errorf("target account %s is not a normal user (uid %d, home %q)", username, uid, fields[5])
		}
		return uid, gid, home, nil
	}
	if err := scanner.Err(); err != nil {
		return 0, 0, "", fmt.Errorf("read target accounts: %w", err)
	}
	return 0, 0, "", fmt.Errorf("user %s missing from %s after installation", username, passwd)
}
