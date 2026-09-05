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

var mappingPattern = regexp.MustCompile(`boot\.initrd\.luks\.devices\.\"?([^\".]+)\"?\.device\s*=\s*\"([^\"]+)\"\s*;`)

type MappingInfo struct {
	Name   string
	Device string
}

func MappingDetails(path string) (MappingInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return MappingInfo{}, err
	}
	match := mappingPattern.FindSubmatch(data)
	if len(match) != 3 {
		return MappingInfo{}, nil
	}
	return MappingInfo{Name: string(match[1]), Device: string(match[2])}, nil
}

func Mapping(path string) (string, error) {
	info, err := MappingDetails(path)
	if err != nil {
		return "", err
	}
	return info.Name, nil
}

func VerifyDeviceIdentity(expected, detected string) error {
	if !strings.HasPrefix(expected, "/dev/disk/by-uuid/") {
		return fmt.Errorf("refusing disk-key operation: configured LUKS device %q is not a stable /dev/disk/by-uuid path", expected)
	}
	expectedResolved, err := filepath.EvalSymlinks(expected)
	if err != nil {
		return fmt.Errorf("resolve configured LUKS device %s: %w", expected, err)
	}
	detectedResolved, err := filepath.EvalSymlinks(detected)
	if err != nil {
		return fmt.Errorf("resolve detected LUKS device %s: %w", detected, err)
	}
	if expectedResolved != detectedResolved {
		return fmt.Errorf("refusing disk-key operation: configured LUKS device %s resolves to %s, but detected %s", expected, expectedResolved, detectedResolved)
	}
	return nil
}

func DetectDevice(ctx context.Context) (string, error) {
	out, err := exec.CommandContext(ctx, "lsblk", "-pnro", "NAME,FSTYPE").Output()
	if err != nil {
		return "", fmt.Errorf("lsblk: %w", err)
	}
	devices := parseLUKSDevices(string(out))
	if len(devices) > 1 {
		return "", fmt.Errorf("multiple LUKS devices detected (%s); refusing ambiguous automatic selection", strings.Join(devices, ", "))
	}
	if len(devices) == 1 {
		return devices[0], nil
	}
	return "", nil
}

func parseLUKSDevices(output string) []string {
	var devices []string
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[1] == "crypto_LUKS" && filepath.IsAbs(fields[0]) {
			devices = append(devices, fields[0])
		}
	}
	return devices
}

func TPMAvailable() bool {
	_, a := os.Stat("/dev/tpmrm0")
	_, b := os.Stat("/dev/tpm0")
	return a == nil || b == nil
}

func TPMPolicyAvailable() error {
	for _, path := range []string{"/var/lib/systemd/pcrlock.json"} {
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("signed TPM policy missing at %s: %w", path, err)
		}
		if !info.Mode().IsRegular() || info.Size() == 0 {
			return fmt.Errorf("signed TPM policy at %s is not a non-empty regular file", path)
		}
	}
	return nil
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
	if err := TPMPolicyAvailable(); err != nil {
		return fmt.Errorf("refusing unbound TPM enrollment: %w", err)
	}
	key, cleanup, err := keyFile(passphrase)
	if err != nil {
		return err
	}
	defer cleanup()
	args := append([]string{"systemd-cryptenroll"}, enrollmentArgs(device, key)...)
	return attached(ctx, "sudo", args...)
}

func enrollmentArgs(device, key string) []string {
	return []string{
		"--unlock-key-file=" + key,
		"--tpm2-device=auto",
		"--tpm2-pcrlock=/var/lib/systemd/pcrlock.json",
		device,
	}
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

func AddKeysPreservingExisting(ctx context.Context, device, oldKey, newKey, recoveryKey string) error {
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
	if err := attached(ctx, "sudo", "cryptsetup", "open", "--test-passphrase", "--type", "luks", device, "--key-file", newPath); err != nil {
		return fmt.Errorf("new LUKS key was enrolled but verification failed; no existing key was removed: %w", err)
	}
	if err := attached(ctx, "sudo", "cryptsetup", "open", "--test-passphrase", "--type", "luks", device, "--key-file", recoveryPath); err != nil {
		return fmt.Errorf("recovery key was enrolled but verification failed; no existing key was removed: %w", err)
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
