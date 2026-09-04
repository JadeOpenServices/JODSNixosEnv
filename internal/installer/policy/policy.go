// Package policy derives non-interactive installer choices from typed input.
package policy

import "github.com/bakanura/gjallarOS/internal/installer/config"

type Features struct {
	AIEnable               bool
	AutoReboot             bool
	DebugFunctions         bool
	DockerEnable           bool
	ClamshellEnable        bool
	USBGuardEnable         bool
	NemuEnable             bool
	NemuGPUPassthrough     bool
	RecoveryEnable         bool
	JODSPrebootLockEnable  bool
	SecureBootEnable       bool
	EndpointManagedDevice  bool
	TouchpadWorkspaceSwipe bool
	WorkUserEnable         bool
}

func FromUser(user config.User) Features {
	return Features{
		AIEnable: user.AIEnable, AutoReboot: user.AutoReboot,
		DebugFunctions: user.DebugFunctions, DockerEnable: user.DockerEnable,
		ClamshellEnable: user.ClamshellEnable, USBGuardEnable: user.USBGuardEnable,
		NemuEnable: user.NemuEnable, NemuGPUPassthrough: user.NemuGPUPassthrough,
		RecoveryEnable: user.RecoveryEnable, JODSPrebootLockEnable: user.JODSPrebootLockEnable,
		SecureBootEnable:       user.SecureBootEnable,
		EndpointManagedDevice:  user.EndpointManagedDevice,
		TouchpadWorkspaceSwipe: user.TouchpadWorkspaceSwipe, WorkUserEnable: user.WorkUserEnable,
	}
}
