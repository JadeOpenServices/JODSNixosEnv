package policy

import (
	"github.com/bakanura/gjallarOS/internal/installer/config"
	"testing"
)

func TestFromUser(t *testing.T) {
	got := FromUser(config.User{AIEnable: true, USBGuardEnable: true, NemuGPUPassthrough: true})
	if !got.AIEnable || !got.USBGuardEnable || !got.NemuGPUPassthrough || got.DockerEnable {
		t.Fatalf("unexpected features: %#v", got)
	}
}
