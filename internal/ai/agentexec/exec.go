// Package agentexec executes only named, policy-classified capabilities.
package agentexec

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	policy "github.com/bakanura/gjallarOS/internal/ai/policy"
)

type Request struct {
	Workspace, Tool               string
	Args                          []string
	Mode, ApprovalPath, AuditPath string
	Timeout                       time.Duration
}
type auditEvent struct {
	Timestamp                                   time.Time `json:"timestamp"`
	Tool, Workspace, Capability, Policy, Result string
	Risk                                        int `json:"risk"`
	ApprovalRequired, AutoApproved              bool
	ExitCode                                    int `json:"exitCode"`
}

func Run(ctx context.Context, r Request, stdout, stderr io.Writer) error {
	workspace, err := filepath.Abs(r.Workspace)
	if err != nil {
		return err
	}
	p, err := policy.DefaultProfile(r.Mode)
	if err != nil {
		return err
	}
	d := policy.Decide(p, workspace, r.Tool, r.Args)
	e := auditEvent{Timestamp: time.Now().UTC(), Tool: r.Tool, Workspace: workspace, Capability: d.Capability, Policy: p.Mode, Risk: int(d.Risk), ApprovalRequired: d.Approval}
	if !d.Allowed {
		if !d.Approval {
			e.Result = "denied"
			appendAudit(r.AuditPath, e)
			return fmt.Errorf("policy denied %s: %s", r.Tool, d.Reason)
		}
		if err := policy.ConsumeGrant(r.ApprovalPath, workspace, r.Tool, r.Args, time.Now()); err != nil {
			e.Result = "approval-required"
			appendAudit(r.AuditPath, e)
			return err
		}
	} else if d.Risk >= policy.Privileged {
		e.AutoApproved = true
	}
	name, args, err := command(workspace, r.Tool, r.Args)
	if err != nil {
		return err
	}
	if r.Timeout <= 0 || r.Timeout > 30*time.Minute {
		r.Timeout = 10 * time.Minute
	}
	bounded, cancel := context.WithTimeout(ctx, r.Timeout)
	defer cancel()
	cmd := exec.CommandContext(bounded, name, args...)
	cmd.Dir = workspace
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Env = []string{"PATH=/run/current-system/sw/bin:/etc/profiles/per-user/" + os.Getenv("USER") + "/bin", "HOME=" + os.Getenv("HOME"), "LANG=C.UTF-8", "GIT_TERMINAL_PROMPT=0", "NIX_CONFIG=accept-flake-config = false"}
	err = cmd.Run()
	e.ExitCode = exitCode(err)
	if bounded.Err() != nil {
		e.Result = "timeout"
		appendAudit(r.AuditPath, e)
		return fmt.Errorf("%s exceeded %s", r.Tool, r.Timeout)
	}
	if err != nil {
		e.Result = "failed"
	} else {
		e.Result = "ok"
	}
	appendAudit(r.AuditPath, e)
	return err
}

func command(workspace, tool string, a []string) (string, []string, error) {
	switch tool {
	case "git-inspect":
		if len(a) == 0 || !oneOf(a[0], "status", "diff", "log", "show", "branch") {
			return "", nil, fmt.Errorf("git-inspect requires status, diff, log, show, or branch")
		}
		return "git", append([]string{"-C", workspace, "--no-pager"}, a...), nil
	case "repo-search":
		return "rg", append([]string{"--", strings.Join(a, " "), workspace}, nil...), nil
	case "repo-test":
		return "go", []string{"test", "./..."}, nil
	case "repo-format":
		return "nixfmt", []string{"--check", workspace}, nil
	case "nix-eval":
		return "nix", append([]string{"eval", "--no-write-lock-file"}, a...), nil
	case "nix-check":
		return "nix", append([]string{"flake", "check", "--no-build", "--no-write-lock-file", "path:" + workspace}, a...), nil
	case "nix-build":
		return "nix", append([]string{"build", "--no-link", "--no-write-lock-file"}, a...), nil
	case "nixos-dry-build":
		return "nixos-rebuild", append([]string{"dry-build", "--flake"}, a...), nil
	case "system-inspect":
		if len(a) == 0 || !oneOf(a[0], "status", "show", "is-active", "is-failed") {
			return "", nil, fmt.Errorf("system-inspect accepts bounded systemctl queries")
		}
		return "systemctl", a, nil
	case "local-command":
		if len(a) == 0 || strings.ContainsRune(a[0], filepath.Separator) {
			return "", nil, fmt.Errorf("local-command requires a PATH-resolved executable and argv")
		}
		return a[0], a[1:], nil
	case "service-manage":
		if len(a) != 2 || !oneOf(a[0], "start", "stop", "restart") {
			return "", nil, fmt.Errorf("service-manage requires ACTION UNIT")
		}
		return "systemctl", a, nil
	case "nixos-deploy":
		if len(a) < 2 || !oneOf(a[0], "switch", "boot") {
			return "", nil, fmt.Errorf("nixos-deploy requires switch|boot and flake target")
		}
		return "nixos-rebuild", append([]string{a[0], "--flake"}, a[1:]...), nil
	default:
		return "", nil, fmt.Errorf("unknown tool %q", tool)
	}
}

func oneOf(s string, values ...string) bool {
	for _, v := range values {
		if s == v {
			return true
		}
	}
	return false
}
func exitCode(err error) int {
	if err == nil {
		return 0
	}
	if e, ok := err.(*exec.ExitError); ok {
		return e.ExitCode()
	}
	return -1
}
func appendAudit(path string, e auditEvent) {
	if path == "" {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0700)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer f.Close()
	b, _ := json.Marshal(e)
	_, _ = f.Write(append(b, '\n'))
}
