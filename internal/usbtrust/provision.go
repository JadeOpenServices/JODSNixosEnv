package usbtrust

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ProvisionTPM explicitly creates a non-exportable P-256 signing primary at
// an unoccupied owner persistent handle. It never evicts an existing object.
func ProvisionTPM(ctx context.Context, dir, handle string) error {
	signer, err := NewTPMSigner(handle)
	if err != nil {
		return err
	}
	return provisionTPM(ctx, dir, handle, signer, func(ctx context.Context, name string, args ...string) ([]byte, error) {
		return exec.CommandContext(ctx, name, args...).CombinedOutput()
	})
}

type provisionCommand func(context.Context, string, ...string) ([]byte, error)

func provisionTPM(ctx context.Context, dir, handle string, signer Signer, command provisionCommand) (result error) {
	if !strings.HasPrefix(strings.ToLower(handle), "0x810") {
		return fmt.Errorf("USB trust requires an owner persistent handle in 0x81000000..0x810fffff")
	}
	if present, err := SignedStatePresent(dir); err != nil {
		return err
	} else if present {
		return fmt.Errorf("refusing to provision a new key over existing USB trust state")
	}
	output, err := command(ctx, "tpm2_getcap", "handles-persistent")
	if err != nil {
		return fmt.Errorf("inspect TPM handles: %w: %s", err, output)
	}
	if strings.Contains(strings.ToLower(string(output)), strings.ToLower(handle)) {
		return fmt.Errorf("TPM handle %s is occupied; choose an unused handle", handle)
	}
	tmp, err := os.MkdirTemp("", "gjallar-usbtrust-provision-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	object := filepath.Join(tmp, "signing.ctx")
	run := func(name string, args ...string) error {
		output, err := command(ctx, name, args...)
		if err != nil {
			return fmt.Errorf("%s: %w: %s", name, err, output)
		}
		return nil
	}
	if err := run("tpm2_createprimary", "-Q", "-C", "o", "-G", "ecc256:ecdsa-sha256", "-g", "sha256", "-a", "fixedtpm|fixedparent|sensitivedataorigin|userwithauth|sign", "-c", object); err != nil {
		return err
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = command(cleanup, "tpm2_flushcontext", object)
	}()
	if err := run("tpm2_evictcontrol", "-Q", "-C", "o", "-c", object, handle); err != nil {
		return err
	}
	committed := false
	defer func() {
		if committed {
			return
		}
		cleanup, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		output, err := command(cleanup, "tpm2_evictcontrol", "-Q", "-C", "o", "-c", handle)
		if err != nil {
			result = errors.Join(result, fmt.Errorf("rollback USB trust key at %s failed; handle may remain occupied: %w: %s", handle, err, output))
		}
	}()
	if err := ValidateSigner(ctx, signer); err != nil {
		return fmt.Errorf("key provisioned at %s but self-test failed: %w", handle, err)
	}
	committed = true
	return nil
}
