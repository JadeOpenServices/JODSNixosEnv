// Package policy derives non-interactive installer choices from typed input.
package policy

import "github.com/bakanura/gjallarOS/internal/installer/config"

type Features struct {
	AIEnable               bool
	AutoReboot             bool
	DebugFunctions         bool
	ContainersEnable       bool
	ClamshellEnable        bool
	USBGuardEnable         bool
	USBTrustEnforce        bool
	USBTrustTPMHandle      string
	NemuEnable             bool
	RecoveryEnable         bool
	JODSPrebootLockEnable  bool
	SecureBootEnable       bool
	EndpointManagedDevice  bool
	TouchpadWorkspaceSwipe bool
	PrintingEnable         bool
	NetworkPrintingEnable  bool
}

func FromUser(user config.User) Features {
	return Features{
		AIEnable: user.AIEnable, AutoReboot: user.AutoReboot,
		DebugFunctions: user.DebugFunctions, ContainersEnable: user.ContainersEnable,
		ClamshellEnable: user.ClamshellEnable, USBGuardEnable: user.USBGuardEnable, USBTrustEnforce: user.USBTrustEnforce, USBTrustTPMHandle: user.USBTrustTPMHandle, PrintingEnable: user.PrintingEnable, NetworkPrintingEnable: user.NetworkPrintingEnable,
		NemuEnable:     user.NemuEnable,
		RecoveryEnable: user.RecoveryEnable, JODSPrebootLockEnable: user.JODSPrebootLockEnable,
		SecureBootEnable:       user.SecureBootEnable,
		EndpointManagedDevice:  user.EndpointManagedDevice,
		TouchpadWorkspaceSwipe: user.TouchpadWorkspaceSwipe,
	}
}
