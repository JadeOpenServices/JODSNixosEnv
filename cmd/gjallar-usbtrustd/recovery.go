package main

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
)

// luksVerifier accepts a passphrase that opens any configured LUKS key slot.
// --test-passphrase never creates a mapping, and --disable-locks keeps the
// check read-only under ProtectSystem=strict.
func luksVerifier(cryptsetup string, devices []string) func(context.Context, []byte) error {
	return func(ctx context.Context, key []byte) error {
		if len(devices) == 0 {
			return errors.New("no LUKS devices are configured")
		}
		for _, device := range devices {
			cmd := exec.CommandContext(ctx, cryptsetup, "open", "--test-passphrase", "--disable-locks", "--key-file=-", device)
			cmd.Stdin = bytes.NewReader(key)
			if cmd.Run() == nil {
				return nil
			}
		}
		return errors.New("no LUKS key slot accepted the passphrase")
	}
}
