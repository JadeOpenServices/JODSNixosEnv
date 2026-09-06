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
		args, err := boundedNixFlakeArgs(workspace, a)
		if err != nil {
			return "", nil, err
		}
		return "nix", append([]string{
			"eval",
			"--no-write-lock-file",
			"--no-update-lock-file",
			"--no-use-registries",
		}, args...), nil
	case "nix-check":
		flags, err := boundedNixFlags(a)
		if err != nil {
			return "", nil, err
		}
		args := []string{
			"flake",
			"check",
			"--no-build",
			"--no-write-lock-file",
			"--no-update-lock-file",
			"--no-use-registries",
			"path:" + workspace,
		}
		return "nix", append(args, flags...), nil
	case "nix-build":
		args, err := boundedNixFlakeArgs(workspace, a)
		if err != nil {
			return "", nil, err
		}
		return "nix", append([]string{
			"build",
			"--no-link",
			"--no-write-lock-file",
			"--no-update-lock-file",
			"--no-use-registries",
		}, args...), nil
	case "nixos-dry-build":
		target, flags, err := boundedRebuildArgs(workspace, a)
		if err != nil {
			return "", nil, err
		}
		args := []string{
			"dry-build",
			"--flake",
			target,
			"--no-write-lock-file",
			"--no-update-lock-file",
		}
		return "nixos-rebuild", append(args, flags...), nil
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
			return "", nil, fmt.Errorf(
				"nixos-deploy requires switch|boot and a workspace flake target",
			)
		}

		target, flags, err := boundedRebuildArgs(workspace, a[1:])
		if err != nil {
			return "", nil, err
		}

		args := []string{
			a[0],
			"--flake",
			target,
			"--no-write-lock-file",
			"--no-update-lock-file",
		}
		return "nixos-rebuild", append(args, flags...), nil
	default:
		return "", nil, fmt.Errorf("unknown tool %q", tool)
	}
}

// workspaceFlakeTarget converts the only accepted AI-visible flake reference
// forms into an explicit path: reference for the active workspace.
func workspaceFlakeTarget(workspace, target string) (string, error) {
	workspace, err := filepath.Abs(workspace)
	if err != nil {
		return "", fmt.Errorf("resolve workspace: %w", err)
	}
	workspace = filepath.Clean(workspace)

	base := "path:" + workspace

	switch {
	case target == "", target == ".":
		return base, nil
	case strings.HasPrefix(target, ".#"):
		return base + target[1:], nil
	case strings.HasPrefix(target, "#"):
		return base + target, nil
	case target == base:
		return base, nil
	case strings.HasPrefix(target, base+"#"):
		return target, nil
	default:
		return "", fmt.Errorf(
			"Nix flake target %q is outside the active workspace",
			target,
		)
	}
}

func boundedNixFlags(args []string) ([]string, error) {
	allowed := map[string]bool{
		"--json":             true,
		"--raw":              true,
		"--show-trace":       true,
		"--keep-going":       true,
		"--print-build-logs": true,
		"--print-out-paths":  true,
		"--quiet":            true,
		"--verbose":          true,
		"-L":                 true,
		"-v":                 true,
	}

	out := make([]string, 0, len(args))

	for _, arg := range args {
		if !allowed[arg] {
			return nil, fmt.Errorf(
				"Nix option %q is not permitted for the workspace agent",
				arg,
			)
		}
		out = append(out, arg)
	}

	return out, nil
}

func boundedNixFlakeArgs(
	workspace string,
	args []string,
) ([]string, error) {
	flags := make([]string, 0, len(args))
	target := ""
	targetSeen := false

	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			validated, err := boundedNixFlags([]string{arg})
			if err != nil {
				return nil, err
			}
			flags = append(flags, validated...)
			continue
		}

		if targetSeen {
			return nil, fmt.Errorf(
				"Nix workspace operation accepts only one flake target",
			)
		}

		var err error
		target, err = workspaceFlakeTarget(workspace, arg)
		if err != nil {
			return nil, err
		}
		targetSeen = true
	}

	if !targetSeen {
		var err error
		target, err = workspaceFlakeTarget(workspace, ".")
		if err != nil {
			return nil, err
		}
	}

	return append(flags, target), nil
}

func boundedRebuildArgs(
	workspace string,
	args []string,
) (string, []string, error) {
	if len(args) == 0 {
		return "", nil, fmt.Errorf(
			"nixos-rebuild requires a workspace flake target",
		)
	}

	target, err := workspaceFlakeTarget(workspace, args[0])
	if err != nil {
		return "", nil, err
	}

	flags, err := boundedNixFlags(args[1:])
	if err != nil {
		return "", nil, err
	}

	return target, flags, nil
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
