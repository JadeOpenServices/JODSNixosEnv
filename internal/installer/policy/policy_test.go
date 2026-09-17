package policy

import (
	"testing"

	"github.com/bakanura/gjallarOS/internal/installer/config"
)

func TestFromUser(t *testing.T) {
	got := FromUser(config.User{
		AIEnable:       true,
		USBGuardEnable: true,
		NemuEnable:     true,
	})

	if !got.AIEnable || !got.USBGuardEnable || !got.NemuEnable || got.ContainersEnable {
		t.Fatalf("unexpected features: %#v", got)
	}
}
