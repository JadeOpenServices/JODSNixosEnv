package recoveryresize

import (
	"context"
	"fmt"
	"strings"
)

type OutputRunner interface {
	Output(ctx context.Context, name string, args ...string) ([]byte, error)
}

// DetectRootFilesystem performs the ticket's actual-filesystem capability gate.
// It intentionally asks the running system rather than trusting diskplan or
// installer configuration.
func DetectRootFilesystem(
	ctx context.Context,
	runner OutputRunner,
	mountpoint string,
) (string, error) {
	mountpoint = strings.TrimSpace(mountpoint)
	if mountpoint == "" {
		return "", fmt.Errorf("root mountpoint is required")
	}

	out, err := runner.Output(
		ctx,
		"findmnt",
		"-nro",
		"FSTYPE",
		"--target",
		mountpoint,
	)
	if err != nil {
		return "", fmt.Errorf(
			"detect actual root filesystem at %s: %w",
			mountpoint,
			err,
		)
	}

	fs := strings.ToLower(strings.TrimSpace(string(out)))
	if fs == "" {
		return "", fmt.Errorf(
			"detect actual root filesystem at %s: empty result",
			mountpoint,
		)
	}

	return fs, nil
}

func RequireBtrfsFilesystem(fs string) error {
	fs = strings.ToLower(strings.TrimSpace(fs))
	if fs != FilesystemBtrfs {
		return fmt.Errorf(
			"%w. Current filesystem: %s. No disk changes were made.",
			ErrUnsupportedFilesystem,
			printableFilesystem(fs),
		)
	}
	return nil
}
