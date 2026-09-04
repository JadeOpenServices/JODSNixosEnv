package diskcrypto

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var mappingPattern = regexp.MustCompile(`boot\.initrd\.luks\.devices\.\"?([^\".]+)\"?\.device`)

func Mapping(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	match := mappingPattern.FindSubmatch(data)
	if len(match) != 2 {
		return "", nil
	}
	return string(match[1]), nil
}

func DetectDevice(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "lsblk", "-pnro", "NAME,FSTYPE").Output()
	if err != nil {
		return "", fmt.Errorf("lsblk: %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == "crypto_LUKS" && filepath.IsAbs(fields[0]) {
			return fields[0], nil
		}
	}
	return "", nil
}

func TPMAvailable() bool {
	_, a := os.Stat("/dev/tpmrm0")
	_, b := os.Stat("/dev/tpm0")
	return a == nil || b == nil
}

func Verify(ctx context.Context, device, passphrase string) error {
	key, cleanup, err := keyFile(passphrase)
	if err != nil {
		return err
	}
	defer cleanup()
	return attached(ctx, "sudo", "cryptsetup", "open", "--test-passphrase", "--type", "luks", device, "--key-file", key)
}

func EnrollTPM(ctx context.Context, device, passphrase string) error {
	key, cleanup, err := keyFile(passphrase)
	if err != nil {
		return err
	}
	defer cleanup()
	return attached(ctx, "sudo", "systemd-cryptenroll", "--unlock-key-file="+key, "--tpm2-device=auto", device)
}

func EnableTPMConfig(path, mapping string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if strings.Contains(string(data), "boot.initrd.systemd.tpm2.enable") {
		return nil
	}
	index := strings.LastIndex(string(data), "}")
	if index < 0 {
		return fmt.Errorf("hardware configuration has no closing brace")
	}
	addition := fmt.Sprintf("    boot.initrd.systemd.enable = true;\n    boot.initrd.systemd.tpm2.enable = true;\n    boot.initrd.luks.devices.%q.crypttabExtraOpts = [ \"tpm2-device=auto\" ];\n", mapping)
	updated := append(append([]byte{}, data[:index]...), append([]byte(addition), data[index:]...)...)
	return atomicWrite(path, updated)
}

type RotationResult struct{ NewKey, RecoveryKey string }

func GenerateKey() (string, error) {
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

func Rotate(ctx context.Context, device, oldKey, newKey, recoveryKey string) error {
	oldPath, oldCleanup, err := keyFile(oldKey)
	if err != nil {
		return err
	}
	defer oldCleanup()
	newPath, newCleanup, err := keyFile(newKey)
	if err != nil {
		return err
	}
	defer newCleanup()
	recoveryPath, recoveryCleanup, err := keyFile(recoveryKey)
	if err != nil {
		return err
	}
	defer recoveryCleanup()
	if err := attached(ctx, "sudo", "cryptsetup", "luksAddKey", device, newPath, "--key-file", oldPath); err != nil {
		return fmt.Errorf("enroll new LUKS key: %w", err)
	}
	if err := attached(ctx, "sudo", "cryptsetup", "luksAddKey", device, recoveryPath, "--key-file", oldPath); err != nil {
		return fmt.Errorf("recovery key enrollment failed; new and original keys remain enrolled: %w", err)
	}
	if err := attached(ctx, "sudo", "cryptsetup", "luksRemoveKey", device, "--key-file", oldPath); err != nil {
		return fmt.Errorf("critical: new and recovery keys were enrolled, but original key removal failed: %w", err)
	}
	return nil
}

func keyFile(secret string) (string, func(), error) {
	f, err := os.CreateTemp("", "gjallar-luks-key-*")
	if err != nil {
		return "", func() {}, err
	}
	name := f.Name()
	cleanup := func() { _ = os.Remove(name) }
	if err := f.Chmod(0600); err != nil {
		f.Close()
		cleanup()
		return "", func() {}, err
	}
	if _, err := f.WriteString(secret); err != nil {
		f.Close()
		cleanup()
		return "", func() {}, err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		cleanup()
		return "", func() {}, err
	}
	if err := f.Close(); err != nil {
		cleanup()
		return "", func() {}, err
	}
	return name, cleanup, nil
}

func atomicWrite(path string, data []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".hardware-tpm-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err := f.Chmod(0644); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}

func attached(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
