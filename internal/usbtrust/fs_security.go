package usbtrust

import (
	"fmt"
	"os"
	"syscall"
)

const (
	StateDirectoryMode os.FileMode = 0700
	StateFileMode      os.FileMode = 0600
)

func EnsureSecureStateDirectory(path string) error {
	info, err := os.Lstat(path)

	if os.IsNotExist(err) {
		if err := os.MkdirAll(path, StateDirectoryMode); err != nil {
			return fmt.Errorf("create USB trust state directory: %w", err)
		}

		info, err = os.Lstat(path)
	}

	if err != nil {
		return fmt.Errorf("inspect USB trust state directory: %w", err)
	}

	return validateOwnedDirectory(
		path,
		info,
		uint32(os.Geteuid()),
	)
}

func ValidateSecureStateFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect USB trust state file: %w", err)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf(
			"USB trust state file %q must not be a symlink",
			path,
		)
	}

	if !info.Mode().IsRegular() {
		return fmt.Errorf(
			"USB trust state file %q is not regular",
			path,
		)
	}

	if info.Mode().Perm() != StateFileMode {
		return fmt.Errorf(
			"USB trust state file %q has permissions %04o, want %04o",
			path,
			info.Mode().Perm(),
			StateFileMode,
		)
	}

	owner, err := ownerUID(info)
	if err != nil {
		return err
	}

	expected := uint32(os.Geteuid())

	if owner != expected {
		return fmt.Errorf(
			"USB trust state file %q owner UID %d, want %d",
			path,
			owner,
			expected,
		)
	}

	return nil
}

func validateOwnedDirectory(
	path string,
	info os.FileInfo,
	expectedUID uint32,
) error {
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf(
			"USB trust state directory %q must not be a symlink",
			path,
		)
	}

	if !info.IsDir() {
		return fmt.Errorf(
			"USB trust state path %q is not a directory",
			path,
		)
	}

	if info.Mode().Perm() != StateDirectoryMode {
		return fmt.Errorf(
			"USB trust state directory %q has permissions %04o, want %04o",
			path,
			info.Mode().Perm(),
			StateDirectoryMode,
		)
	}

	owner, err := ownerUID(info)
	if err != nil {
		return err
	}

	if owner != expectedUID {
		return fmt.Errorf(
			"USB trust state directory %q owner UID %d, want %d",
			path,
			owner,
			expectedUID,
		)
	}

	return nil
}

func ownerUID(info os.FileInfo) (uint32, error) {
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, fmt.Errorf(
			"USB trust filesystem ownership unavailable",
		)
	}

	return stat.Uid, nil
}
