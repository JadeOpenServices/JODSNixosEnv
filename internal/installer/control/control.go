// Package control locates gjallarctl for the installer binaries.
package control

import (
	"os"
	"path/filepath"
)

// Path is gjallarctl's store path, set with -ldflags -X by
// pkgs/gjallar-installer: the installer package no longer ships gjallarctl
// next to its own binaries.
var Path string

// Binary returns Path, else a gjallarctl next to the running executable,
// else "gjallarctl" for a PATH lookup.
func Binary() string {
	if Path != "" {
		return Path
	}
	if exe, err := os.Executable(); err == nil {
		candidate := filepath.Join(filepath.Dir(exe), "gjallarctl")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return "gjallarctl"
}
