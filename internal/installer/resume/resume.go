package resume

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	Schema        = 1
	StateDir      = "/var/lib/gjallarOS/installer-resume"
	PendingPath   = StateDir + "/pending.json"
	ActivePath    = StateDir + "/active.json"
	InstallerPath = StateDir + "/gjallar-installer"
	ControlPath   = StateDir + "/gjallarctl"
	ModulePath    = StateDir + "/resume-module.nix"
	WrapperPath   = StateDir + "/configuration.nix"
	ServiceName   = "gjallar-installer-resume.service"
)

type Transaction struct {
	Schema          int
	State           string
	Reason          string
	ExpectedRelease string
	Repo            string
	Args            []string
	CreatedUTC      string
}

func Load(path string) (Transaction, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Transaction{}, fmt.Errorf("read installer resume transaction: %w", err)
	}

	var tx Transaction
	if err := json.Unmarshal(data, &tx); err != nil {
		return Transaction{}, fmt.Errorf("parse installer resume transaction: %w", err)
	}

	if tx.Schema != Schema {
		return Transaction{}, fmt.Errorf(
			"unsupported installer resume transaction schema %d",
			tx.Schema,
		)
	}

	if tx.State != "waiting-for-release-reboot" {
		return Transaction{}, fmt.Errorf(
			"installer resume transaction is not resumable: %s",
			tx.State,
		)
	}

	if strings.TrimSpace(tx.ExpectedRelease) == "" {
		return Transaction{}, fmt.Errorf(
			"installer resume transaction has no expected release",
		)
	}

	if !filepath.IsAbs(tx.Repo) {
		return Transaction{}, fmt.Errorf(
			"installer resume repository is not absolute: %q",
			tx.Repo,
		)
	}

	return tx, nil
}

func Arm(
	ctx context.Context,
	executable string,
	repo string,
	args []string,
	expectedRelease string,
) error {
	if !filepath.IsAbs(executable) {
		return fmt.Errorf("installer executable must be absolute")
	}

	if !filepath.IsAbs(repo) {
		return fmt.Errorf("installer repository must be absolute")
	}

	if strings.TrimSpace(expectedRelease) == "" {
		return fmt.Errorf("expected release must not be empty")
	}

	tx := Transaction{
		Schema:          Schema,
		State:           "waiting-for-release-reboot",
		Reason:          "nixos-release-alignment",
		ExpectedRelease: expectedRelease,
		Repo:            repo,
		Args:            append([]string(nil), args...),
		CreatedUTC:      time.Now().UTC().Format(time.RFC3339),
	}

	data, err := json.MarshalIndent(tx, "", "  ")
	if err != nil {
		return fmt.Errorf("encode installer resume transaction: %w", err)
	}
	data = append(data, '\n')

	tempDir, err := os.MkdirTemp("", "gjallar-installer-resume-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	transactionTemp := filepath.Join(tempDir, "pending.json")
	unitTemp := filepath.Join(tempDir, ServiceName)

	if err := os.WriteFile(transactionTemp, data, 0600); err != nil {
		return err
	}

	module :=
		"{ lib, ... }:\n" +
			"{\n" +
			"  systemd.services.gjallar-installer-resume = {\n" +
			"    description = \"Resume GjallarOS installer after staged NixOS release boot\";\n" +
			"    wantedBy = [ \"multi-user.target\" ];\n" +
			"    after = [ \"local-fs.target\" ];\n" +
			"    unitConfig.ConditionPathExists = \"" + PendingPath + "\";\n" +
			"    serviceConfig = {\n" +
			"      Type = \"oneshot\";\n" +
			"      UMask = \"0077\";\n" +
			"      TimeoutStartSec = 0;\n" +
			"      ExecStartPre = \"/run/current-system/sw/bin/mv " +
			PendingPath + " " + ActivePath + "\";\n" +
			"      ExecStart = \"" + InstallerPath +
			" --resume-transaction " + ActivePath + "\";\n" +
			"      ExecStartPost = \"/run/current-system/sw/bin/rm -f " +
			ActivePath + "\";\n" +
			"    };\n" +
			"  };\n" +
			"}\n"

	wrapper :=
		"{ ... }:\n" +
			"{\n" +
			"  imports = [\n" +
			"    /etc/nixos/configuration.nix\n" +
			"    " + ModulePath + "\n" +
			"  ];\n" +
			"}\n"

	if err := os.WriteFile(unitTemp, []byte(module), 0644); err != nil {
		return err
	}

	wrapperTemp := filepath.Join(tempDir, "configuration.nix")
	if err := os.WriteFile(wrapperTemp, []byte(wrapper), 0644); err != nil {
		return err
	}

	if err := sudo(ctx, "install", "-d", "-m", "0700", StateDir); err != nil {
		return err
	}

	if err := sudo(
		ctx,
		"install",
		"-m", "0755",
		executable,
		InstallerPath,
	); err != nil {
		return fmt.Errorf("install resumable installer binary: %w", err)
	}

	control := filepath.Join(filepath.Dir(executable), "gjallarctl")
	if info, statErr := os.Stat(control); statErr == nil && !info.IsDir() {
		if err := sudo(
			ctx,
			"install",
			"-m", "0755",
			control,
			ControlPath,
		); err != nil {
			return fmt.Errorf("install resumable gjallarctl: %w", err)
		}
	}

	if err := sudo(
		ctx,
		"install",
		"-m", "0600",
		transactionTemp,
		PendingPath,
	); err != nil {
		return fmt.Errorf("install resume transaction: %w", err)
	}

	if _, err := os.Stat("/etc/nixos/configuration.nix"); err != nil {
		return fmt.Errorf(
			"current NixOS configuration is unavailable at /etc/nixos/configuration.nix: %w",
			err,
		)
	}

	if err := sudo(
		ctx,
		"install",
		"-m", "0644",
		unitTemp,
		ModulePath,
	); err != nil {
		return fmt.Errorf("install resume NixOS module: %w", err)
	}

	if err := sudo(
		ctx,
		"install",
		"-m", "0644",
		wrapperTemp,
		WrapperPath,
	); err != nil {
		return fmt.Errorf("install resume NixOS wrapper: %w", err)
	}

	return nil
}

func sudo(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "sudo", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf(
			"sudo %s: %w",
			strings.Join(args, " "),
			err,
		)
	}

	return nil
}
