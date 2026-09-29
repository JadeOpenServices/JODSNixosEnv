package credential

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var usernamePattern = regexp.MustCompile(`^[a-z_][a-z0-9_-]*$`)

func Path(username string) (string, error) {
	if !usernamePattern.MatchString(username) {
		return "", fmt.Errorf("invalid username: %q", username)
	}
	return filepath.Join("/var/lib/gjallarOS/passwords", username+".hash"), nil
}

func Exists(ctx context.Context, path string) bool {
	return exec.CommandContext(ctx, "sudo", "test", "-s", path).Run() == nil
}

func ReadConfirmedPassword(tty *os.File, out io.Writer, username string) (string, error) {
	reader := bufio.NewReader(tty)
	for {
		first, err := ReadSecret(tty, reader, out, fmt.Sprintf("Password for %s: ", username))
		if err != nil {
			return "", err
		}
		second, err := ReadSecret(tty, reader, out, "Confirm password: ")
		if err != nil {
			return "", err
		}
		if first == "" {
			fmt.Fprintln(out, "Password must not be empty.")
			continue
		}
		if first != second {
			fmt.Fprintln(out, "Passwords do not match.")
			continue
		}
		return first, nil
	}
}

func ReadSecret(tty *os.File, reader *bufio.Reader, out io.Writer, prompt string) (string, error) {
	fmt.Fprint(out, prompt)
	off := exec.Command("stty", "-echo")
	off.Stdin = tty
	if err := off.Run(); err != nil {
		return "", fmt.Errorf("disable terminal echo: %w", err)
	}
	defer func() { on := exec.Command("stty", "echo"); on.Stdin = tty; _ = on.Run() }()
	line, err := reader.ReadString('\n')
	fmt.Fprintln(out)
	return strings.TrimRight(line, "\r\n"), err
}

func Hash(ctx context.Context, password string) (string, error) {
	var cmd *exec.Cmd
	if _, err := exec.LookPath("mkpasswd"); err == nil {
		cmd = exec.CommandContext(ctx, "mkpasswd", "--method=yescrypt", "--stdin")
	} else if _, err := exec.LookPath("openssl"); err == nil {
		cmd = exec.CommandContext(ctx, "openssl", "passwd", "-6", "-stdin")
	} else {
		return "", fmt.Errorf("mkpasswd or openssl is required")
	}
	cmd.Stdin = strings.NewReader(password)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// Apply sets the current password hash of an existing account. With
// users.mutableUsers=true NixOS only uses hashedPasswordFile when it creates an
// account, so a stored hash never reaches an account that already exists.
func Apply(ctx context.Context, username, hash string) error {
	line, err := chpasswdLine(username, hash)
	if err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, "sudo", "chpasswd", "--encrypted")
	cmd.Stdin = strings.NewReader(line)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("apply password hash for %s: %w", username, err)
	}
	return nil
}

func chpasswdLine(username, hash string) (string, error) {
	if !usernamePattern.MatchString(username) {
		return "", fmt.Errorf("invalid username: %q", username)
	}
	if !strings.HasPrefix(hash, "$") || strings.ContainsAny(hash, ":\r\n") {
		return "", fmt.Errorf("refusing malformed password hash for %s", username)
	}
	return username + ":" + hash + "\n", nil
}

func Store(ctx context.Context, target, hash string) error {
	tmp, err := os.CreateTemp("", "gjallar-password-*.hash")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err := tmp.Chmod(0600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := fmt.Fprintln(tmp, hash); err != nil {
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
	for _, args := range storeCommands(name, target) {
		cmd := exec.CommandContext(ctx, "sudo", args...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("store password hash: %w", err)
		}
	}
	return nil
}

// storeCommands installs the hash root-only. `install -D` alone would create
// the password directory 0755 and expose which accounts have stored hashes;
// `install -d` also tightens a directory an earlier version created.
func storeCommands(source, target string) [][]string {
	return [][]string{
		{"install", "-d", "-m", "0700", "-o", "root", "-g", "root", "--", filepath.Dir(target)},
		{"install", "-m", "0600", "-o", "root", "-g", "root", "--", source, target},
	}
}
