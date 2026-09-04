package prompt

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

var ErrCancelled = errors.New("prompt cancelled")

type UI struct {
	Reader *bufio.Reader
	Out    io.Writer
	GTK    bool
}

func New(in io.Reader, out io.Writer) UI {
	_, zenity := exec.LookPath("zenity")
	gtk := zenity == nil && (os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != "") && os.Getenv("SSH_CONNECTION") == ""
	return UI{bufio.NewReader(in), out, gtk}
}

func (u UI) Confirm(ctx context.Context, message string, defaultYes bool) (bool, error) {
	if u.GTK {
		args := []string{"--question", "--title=GjallarOS installer", "--text=" + message, "--ok-label=Yes", "--cancel-label=No"}
		err := exec.CommandContext(ctx, "zenity", args...).Run()
		if err == nil {
			return true, nil
		}
		if exit, ok := err.(*exec.ExitError); ok && exit.ExitCode() == 1 {
			return false, nil
		}
		return false, ErrCancelled
	}
	suffix := " [y/N] "
	if defaultYes {
		suffix = " [Y/n] "
	}
	fmt.Fprint(u.Out, message+suffix)
	line, err := u.Reader.ReadString('\n')
	if err != nil {
		return false, err
	}
	value := strings.ToLower(strings.TrimSpace(line))
	if value == "" {
		return defaultYes, nil
	}
	return value == "y" || value == "yes", nil
}

func (u UI) Value(ctx context.Context, label, def string) (string, error) {
	if u.GTK {
		out, err := exec.CommandContext(ctx, "zenity", "--entry", "--title=GjallarOS installer", "--text="+label, "--entry-text="+def).Output()
		if err != nil {
			return "", ErrCancelled
		}
		return strings.TrimSpace(string(out)), nil
	}
	fmt.Fprintf(u.Out, "%s [%s]: ", label, def)
	line, err := u.Reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		line = def
	}
	return line, nil
}

func (u UI) Choice(ctx context.Context, label, def string, options []string) (string, error) {
	if len(options) == 0 {
		return "", fmt.Errorf("no options for %s", label)
	}
	if u.GTK {
		args := []string{"--list", "--radiolist", "--title=GjallarOS installer", "--text=" + label, "--column=Selected", "--column=Value"}
		for _, v := range options {
			args = append(args, strconv.FormatBool(v == def), v)
		}
		out, err := exec.CommandContext(ctx, "zenity", args...).Output()
		if err != nil {
			return "", ErrCancelled
		}
		value := strings.TrimSpace(string(out))
		if value == "" {
			return "", ErrCancelled
		}
		return value, nil
	}
	for i, v := range options {
		fmt.Fprintf(u.Out, "  %d) %s\n", i+1, v)
	}
	for {
		value, err := u.Value(ctx, label, def)
		if err != nil {
			return "", err
		}
		for _, candidate := range options {
			if value == candidate {
				return value, nil
			}
		}
		if n, e := strconv.Atoi(value); e == nil && n > 0 && n <= len(options) {
			return options[n-1], nil
		}
		fmt.Fprintln(u.Out, "Choose one listed value.")
	}
}

func (u UI) Multi(ctx context.Context, label string, defaults, options []string) ([]string, error) {
	selected := map[string]bool{}
	for _, v := range defaults {
		selected[v] = true
	}
	if u.GTK {
		args := []string{"--list", "--checklist", "--title=GjallarOS installer", "--text=" + label, "--separator=\n", "--column=Selected", "--column=Value"}
		for _, v := range options {
			args = append(args, strconv.FormatBool(selected[v]), v)
		}
		out, err := exec.CommandContext(ctx, "zenity", args...).Output()
		if err != nil {
			return nil, ErrCancelled
		}
		values := strings.Fields(string(out))
		if len(values) == 0 {
			return nil, fmt.Errorf("select at least one value")
		}
		return values, nil
	}
	fmt.Fprintf(u.Out, "%s (comma-separated) [%s]: ", label, strings.Join(defaults, ","))
	line, err := u.Reader.ReadString('\n')
	if err != nil {
		return nil, err
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return defaults, nil
	}
	allowed := map[string]bool{}
	for _, v := range options {
		allowed[v] = true
	}
	values := []string{}
	for _, v := range strings.Split(line, ",") {
		v = strings.TrimSpace(v)
		if !allowed[v] {
			return nil, fmt.Errorf("unsupported selection %q", v)
		}
		values = append(values, v)
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("select at least one value")
	}
	return values, nil
}
