package app

import (
	"testing"

	"github.com/bakanura/gjallarOS/internal/installer/discovery"
)

func TestMachineProfileForHardware(t *testing.T) {
	tests := []struct {
		name string
		hw   discovery.Hardware
		want string
	}{
		{"desktop", discovery.Hardware{FormFactor: "desktop"}, "desktop"},
		{"laptop", discovery.Hardware{FormFactor: "laptop", LaptopVendor: "generic"}, "laptop"},
		{"framework", discovery.Hardware{FormFactor: "laptop", LaptopVendor: "framework"}, "laptop"},
		{"unknown", discovery.Hardware{}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := machineProfileForHardware(tt.hw); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestContainsValue(t *testing.T) {
	if !containsValue([]string{"desktop", "laptop"}, "desktop") {
		t.Fatal("desktop profile was not found")
	}
	if containsValue([]string{"desktop", "laptop"}, "auto") {
		t.Fatal("auto incorrectly treated as an installable profile")
	}
}
