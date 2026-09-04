package bootstrap

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Plan struct {
	Packages, Missing []string
	Updated           []byte
}

func Build(config []byte, preset bool) Plan {
	requirements := map[string]string{"lspci": "pciutils", "git": "git", "fwupdmgr": "fwupd", "curl": "curl"}
	if !preset {
		requirements["zenity"] = "zenity"
	}
	packages := map[string]bool{}
	missing := []string{}
	for command, pkg := range requirements {
		if _, err := exec.LookPath(command); err != nil {
			packages[pkg] = true
			missing = append(missing, command)
		}
	}
	text := string(config)
	if !strings.Contains(text, "catppuccin-gtk") {
		packages["catppuccin-gtk"] = true
		missing = append(missing, "catppuccin-gtk theme")
	}
	if !strings.Contains(text, "GTK_THEME") {
		packages["catppuccin-gtk"] = true
		missing = append(missing, "GTK_THEME")
	}
	if !regexp.MustCompile(`services\.fwupd\.enable\s*=\s*true`).MatchString(text) {
		packages["fwupd"] = true
		missing = append(missing, "services.fwupd.enable")
	}
	list := make([]string, 0, len(packages))
	for pkg := range packages {
		list = append(list, pkg)
	}
	sort.Strings(list)
	sort.Strings(missing)
	return Plan{Packages: list, Missing: missing, Updated: Transform(config, list)}
}

func Transform(config []byte, packages []string) []byte {
	text := string(config)
	lines := strings.Split(text, "\n")
	start, end := -1, -1
	for i, line := range lines {
		if regexp.MustCompile(`^\s*environment\.systemPackages\s*=`).MatchString(line) {
			start = i
			end = i
			if !strings.Contains(line, "];") {
				for j := i + 1; j < len(lines); j++ {
					if regexp.MustCompile(`^\s*\]\s*;`).MatchString(lines[j]) {
						end = j
						break
					}
				}
			}
			break
		}
	}
	missing := []string{}
	region := ""
	if start >= 0 && end >= start {
		region = strings.Join(lines[start:end+1], " ")
	}
	for _, pkg := range packages {
		if !regexp.MustCompile(`(^|\s)` + regexp.QuoteMeta(pkg) + `(\s|\]|$)`).MatchString(region) {
			missing = append(missing, pkg)
		}
	}
	if len(missing) > 0 {
		if start < 0 {
			at := lastBrace(lines)
			if at >= 0 {
				block := []string{"    environment.systemPackages = with pkgs; ["}
				for _, pkg := range missing {
					block = append(block, "      "+pkg)
				}
				block = append(block, "    ];")
				lines = insert(lines, at, block...)
			}
		} else if start == end {
			lines[start] = strings.Replace(lines[start], "]", " "+strings.Join(missing, " ")+" ]", 1)
		} else {
			addition := []string{}
			for _, pkg := range missing {
				addition = append(addition, "      "+pkg)
			}
			lines = insert(lines, end, addition...)
		}
	}
	text = strings.Join(lines, "\n")
	if !regexp.MustCompile(`services\.fwupd\.enable\s*=\s*true`).MatchString(text) {
		text = insertAfterOpening(text, "    services.fwupd.enable = true;\n")
	}
	gtk := regexp.MustCompile(`GTK_THEME\s*=\s*"[^"]*"`)
	if gtk.MatchString(text) {
		text = gtk.ReplaceAllString(text, `GTK_THEME = "Adwaita:dark"`)
	} else {
		text = insertAfterOpening(text, "    environment.variables.GTK_THEME = \"Adwaita:dark\";\n")
	}
	return []byte(text)
}

func Apply(ctx context.Context, configPath string, updated []byte, now time.Time) (string, error) {
	tmp, err := os.CreateTemp("", "gjallar-bootstrap-*.nix")
	if err != nil {
		return "", err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := tmp.Write(updated); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	backup := configPath + ".gjallar-backup." + now.Format("20060102150405")
	if err := run(ctx, "sudo", "cp", "-a", "--", configPath, backup); err != nil {
		return "", err
	}
	if err := run(ctx, "sudo", "install", "-m", "0644", name, configPath); err != nil {
		return backup, err
	}
	if err := run(ctx, "sudo", "nixos-rebuild", "switch"); err != nil {
		if restore := run(ctx, "sudo", "cp", "-a", "--", backup, configPath); restore != nil {
			return backup, fmt.Errorf("rebuild failed: %v; restore failed: %v", err, restore)
		}
		_ = run(ctx, "sudo", "rm", "-f", "--", backup)
		return "", fmt.Errorf("rebuild failed; previous configuration restored: %w", err)
	}
	if err := run(ctx, "sudo", "rm", "-f", "--", backup); err != nil {
		return backup, fmt.Errorf("rebuild succeeded but backup cleanup failed: %w", err)
	}
	return "", nil
}

func lastBrace(lines []string) int {
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.TrimSpace(lines[i]) == "}" {
			return i
		}
	}
	return -1
}
func insert(lines []string, at int, values ...string) []string {
	out := make([]string, 0, len(lines)+len(values))
	out = append(out, lines[:at]...)
	out = append(out, values...)
	out = append(out, lines[at:]...)
	return out
}
func insertAfterOpening(text, value string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		trim := strings.TrimSpace(line)
		if trim == "{" || strings.HasSuffix(trim, ": {") {
			return strings.Join(insert(lines, i+1, strings.TrimSuffix(value, "\n")), "\n")
		}
	}
	return text
}
func run(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
