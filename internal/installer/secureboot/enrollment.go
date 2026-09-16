package secureboot

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type EnrollmentResult string

const (
	EnrollmentWaitingForSetupMode EnrollmentResult = "waiting-for-setup-mode"
	EnrollmentCompleted           EnrollmentResult = "completed"
)

type enrollmentOps struct {
	present         func(string) (bool, error)
	payload         func(string) ([]byte, bool, error)
	inspect         func(context.Context) (Inspection, error)
	command         func(context.Context, string, ...string) error
	verifyArtifacts func(context.Context) error
	recordOwnership func(context.Context, string) error
	glob            func(string) ([]string, error)
}

func EnrollFirmware(ctx context.Context) (EnrollmentResult, error) {
	if os.Geteuid() != 0 {
		return "", fmt.Errorf("Secure Boot firmware enrollment requires root")
	}

	snapshot, err := LoadFirmwarePolicySnapshot(FirmwarePolicyPath)
	if err != nil {
		return "", fmt.Errorf("load trusted Secure Boot firmware policy: %w", err)
	}

	ops := enrollmentOps{
		present: func(name string) (bool, error) {
			_, present, err := efivarPayload(name)
			return present, err
		},
		payload:         efivarPayload,
		inspect:         Inspect,
		command:         run,
		verifyArtifacts: verifyBootArtifacts,
		recordOwnership: RecordOwnership,
		glob:            filepath.Glob,
	}

	return enrollFirmware(ctx, snapshot, ops)
}

func enrollFirmware(
	ctx context.Context,
	snapshot FirmwarePolicySnapshot,
	ops enrollmentOps,
) (EnrollmentResult, error) {
	if err := ValidateFirmwarePolicySnapshot(snapshot); err != nil {
		return "", fmt.Errorf("invalid trusted Secure Boot firmware policy: %w", err)
	}

	inspection, err := ops.inspect(ctx)
	if err != nil {
		return "", fmt.Errorf("inspect Secure Boot ownership: %w", err)
	}

	if !inspection.SetupMode {
		return EnrollmentWaitingForSetupMode, nil
	}

	if inspection.State != StatePendingEnrollment {
		return "", fmt.Errorf(
			"refusing Secure Boot enrollment in Setup Mode from ownership state %q: %s",
			inspection.State,
			inspection.Description,
		)
	}

	for _, name := range snapshot.RequiredPresent {
		present, err := ops.present(name)
		if err != nil {
			return "", fmt.Errorf("inspect required EFI variable %s: %w", name, err)
		}
		if !present {
			return "", fmt.Errorf(
				"required EFI variable %s is missing; refusing Secure Boot enrollment",
				name,
			)
		}
	}

	for _, name := range snapshot.RequiredAbsent {
		present, err := ops.present(name)
		if err != nil {
			return "", fmt.Errorf("inspect forbidden EFI variable %s: %w", name, err)
		}
		if present {
			return "", fmt.Errorf(
				"EFI variable %s must be absent before Secure Boot enrollment",
				name,
			)
		}
	}

	untouchedBefore := make(map[string][]byte, len(snapshot.Untouched))
	for _, name := range snapshot.Untouched {
		payload, present, err := ops.payload(name)
		if err != nil {
			return "", fmt.Errorf(
				"read untouched EFI variable %s before enrollment: %w",
				name,
				err,
			)
		}
		if !present {
			return "", fmt.Errorf(
				"untouched EFI variable %s is missing before enrollment",
				name,
			)
		}

		untouchedBefore[name] = append([]byte(nil), payload...)
	}

	for _, name := range snapshot.PreserveFirmwareBuiltin {
		matches, err := ops.glob("/sys/firmware/efi/efivars/" + name + "-*")
		if err != nil {
			return "", fmt.Errorf("resolve EFI variable %s: %w", name, err)
		}
		if len(matches) != 1 {
			return "", fmt.Errorf(
				"expected exactly one %s EFI variable before enrollment, found %d",
				name,
				len(matches),
			)
		}
		if err := ops.command(ctx, "chattr", "-i", matches[0]); err != nil {
			return "", fmt.Errorf(
				"clear immutable protection on preserved EFI variable %s: %w",
				name,
				err,
			)
		}
	}

	args := []string{"enroll-keys"}
	if len(snapshot.PreserveFirmwareBuiltin) != 0 {
		args = append(
			args,
			"--firmware-builtin="+strings.Join(snapshot.PreserveFirmwareBuiltin, ","),
		)
	}

	if err := ops.command(ctx, "sbctl", args...); err != nil {
		return "", fmt.Errorf("enroll Secure Boot keys: %w", err)
	}

	for _, name := range snapshot.Untouched {
		payload, present, err := ops.payload(name)
		if err != nil {
			return "", fmt.Errorf(
				"read untouched EFI variable %s after enrollment: %w",
				name,
				err,
			)
		}
		if !present {
			return "", fmt.Errorf(
				"untouched EFI variable %s disappeared during enrollment",
				name,
			)
		}
		if !bytes.Equal(untouchedBefore[name], payload) {
			return "", fmt.Errorf(
				"untouched EFI variable %s changed during enrollment",
				name,
			)
		}
	}

	if err := ops.verifyArtifacts(ctx); err != nil {
		return "", fmt.Errorf("verify Secure Boot artifacts after enrollment: %w", err)
	}

	after, err := ops.inspect(ctx)
	if err != nil {
		return "", fmt.Errorf("inspect Secure Boot ownership after enrollment: %w", err)
	}
	if after.SetupMode {
		return "", fmt.Errorf("Secure Boot enrollment completed without leaving Setup Mode")
	}
	if after.State != StateGjallarManaged {
		return "", fmt.Errorf(
			"Secure Boot enrollment did not establish GjallarOS ownership: %s",
			after.Description,
		)
	}

	if err := ops.recordOwnership(ctx, "enrolled"); err != nil {
		return "", fmt.Errorf("record enrolled Secure Boot ownership: %w", err)
	}

	if err := ops.command(
		ctx,
		"install",
		"-m", "0600",
		"/dev/null",
		FinalMarkerPath,
	); err != nil {
		return "", fmt.Errorf("arm final Secure Boot verification: %w", err)
	}

	if err := ops.command(
		ctx,
		"rm", "-f", "--",
		EnrollmentMarkerPath,
		RecoveryConfirmedMarkerPath,
		RecoveryPassphrasePath,
	); err != nil {
		return "", fmt.Errorf("complete Secure Boot enrollment transaction: %w", err)
	}

	return EnrollmentCompleted, nil
}
