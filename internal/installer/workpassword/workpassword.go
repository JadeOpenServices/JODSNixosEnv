package workpassword

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
		return "", fmt.Errorf("invalid work-account username: %q", username)
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
	cmd := exec.CommandContext(ctx, "sudo", "install", "-D", "-m", "0600", name, target)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("store password hash: %w", err)
	}
	return nil
}
