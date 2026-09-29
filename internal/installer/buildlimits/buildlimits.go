// Package buildlimits caps Nix build parallelism on low-memory machines.
//
// Nix defaults to max-jobs=auto and cores=0, so an 8 GiB machine without swap
// can run nproc parallel builds each spawning nproc compilers. Packages missing
// from the binary cache (for example konsole) are then OOM-killed mid-build.
package buildlimits

import (
	"bufio"
	"bytes"
	"os"
	"runtime"
	"strconv"
	"strings"
)

const (
	gib = uint64(1) << 30
	// memoryPerCompiler is the budget for one C++ compiler process.
	memoryPerCompiler = 2 * gib
	// unlimitedAt keeps Nix defaults once RAM plus swap reaches this size.
	unlimitedAt = 16 * gib
)

// Args returns nixos-rebuild/nixos-install flags for this machine, or nil when
// Nix defaults are safe or memory cannot be read.
func Args() []string {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return nil
	}
	return argsFor(memoryFromMeminfo(data), runtime.NumCPU())
}

func argsFor(memory uint64, cpus int) []string {
	if memory == 0 || memory >= unlimitedAt {
		return nil
	}
	cores := int(memory / memoryPerCompiler)
	if cores > cpus {
		cores = cpus
	}
	if cores < 1 {
		cores = 1
	}
	return []string{"--max-jobs", "1", "--cores", strconv.Itoa(cores)}
}

// memoryFromMeminfo returns MemTotal plus SwapTotal minus Shmem in bytes.
// Shmem is RAM held by tmpfs; on live media that is the writable Nix store
// and the checkout, which compilers cannot use. Without subtracting it, a
// 12 GiB live machine with the temporary 8 GiB install swap counted as
// unlimited and nixos-install was OOM-killed compiling KDE packages.
func memoryFromMeminfo(data []byte) uint64 {
	var total, shmem uint64
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "MemTotal:", "SwapTotal:", "Shmem:":
		default:
			continue
		}
		kib, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			return 0
		}
		if fields[0] == "Shmem:" {
			shmem = kib << 10
		} else {
			total += kib << 10
		}
	}
	if shmem >= total {
		return 1 // still limited: argsFor treats 0 as unknown
	}
	return total - shmem
}
