package firmware

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func Available() bool { _, err := exec.LookPath("fwupdmgr"); return err == nil }

func EnsureService(ctx context.Context) error {
	if !Available() {
		return nil
	}
	cmd := exec.CommandContext(ctx, "sudo", "systemctl", "start", "fwupd.service")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("start fwupd.service: %w", err)
	}
	// fwupd is commonly socket-activated/static, so enable failure is non-fatal.
	_ = exec.CommandContext(ctx, "sudo", "systemctl", "enable", "fwupd.service").Run()
	return nil
}

func Update(ctx context.Context) (string, error) {
	if err := attached(ctx, "sudo", "fwupdmgr", "refresh", "--force"); err != nil {
		return "", fmt.Errorf("refresh firmware metadata: %w", err)
	}
	cmd := exec.CommandContext(ctx, "sudo", "fwupdmgr", "update")
	cmd.Stdin = os.Stdin
	out, err := cmd.CombinedOutput()
	os.Stdout.Write(out)
	if err != nil {
		return string(out), fmt.Errorf("apply firmware updates: %w", err)
	}
	lower := strings.ToLower(string(out))
	noUpdate := strings.Contains(lower, "no updatable") || strings.Contains(lower, "latest available firmware version") || strings.Contains(lower, "no available firmware")
	updated := strings.Contains(lower, "successfully installed") || strings.Contains(lower, "successfully updated") || strings.Contains(lower, "firmware updated")
	if noUpdate && !updated {
		return "none", nil
	}
	return "completed", nil
}

func attached(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
