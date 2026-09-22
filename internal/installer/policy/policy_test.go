package policy

import (
	"testing"

	"github.com/bakanura/gjallarOS/internal/installer/config"
)

func TestFromUser(t *testing.T) {
	got := FromUser(config.User{
		AIEnable:          true,
		USBGuardEnable:    true,
		USBTrustEnforce:   true,
		USBTrustTPMHandle: "0x81000042",
		NemuEnable:        true,
	})

	if !got.AIEnable || !got.USBGuardEnable || !got.USBTrustEnforce || got.USBTrustTPMHandle != "0x81000042" || !got.NemuEnable || got.ContainersEnable {
		t.Fatalf("unexpected features: %#v", got)
	}
}
