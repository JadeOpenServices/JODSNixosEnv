//go:build linux

package broker

import (
	"fmt"
	"net"
	"syscall"
)

func PeerUID(conn *net.UnixConn) (uint32, error) {
	if conn == nil {
		return 0, fmt.Errorf(
			"USB trust peer connection is nil",
		)
	}

	raw, err := conn.SyscallConn()
	if err != nil {
		return 0, fmt.Errorf(
			"USB trust peer syscall connection: %w",
			err,
		)
	}

	var (
		credential *syscall.Ucred
		controlErr error
	)

	if err := raw.Control(func(fd uintptr) {
		credential, controlErr = syscall.GetsockoptUcred(
			int(fd),
			syscall.SOL_SOCKET,
			syscall.SO_PEERCRED,
		)
	}); err != nil {
		return 0, fmt.Errorf(
			"USB trust peer control: %w",
			err,
		)
	}

	if controlErr != nil {
		return 0, fmt.Errorf(
			"USB trust peer credentials: %w",
			controlErr,
		)
	}

	if credential == nil {
		return 0, fmt.Errorf(
			"USB trust peer credentials unavailable",
		)
	}

	return credential.Uid, nil
}
