package deviceprobe

import (
	"strings"

	"github.com/bakanura/gjallarOS/internal/hardware/orientation"
)

type CapabilityState struct {
	Present bool   `json:"present"`
	Details string `json:"details,omitempty"`
}

type FingerprintCapability struct {
	Present           bool `json:"present"`
	UpstreamSupported bool `json:"upstreamSupported"`
}

type Capabilities struct {
	Thunderbolt CapabilityState       `json:"thunderbolt"`
	CardReader  CapabilityState       `json:"cardReader"`
	Fingerprint FingerprintCapability `json:"fingerprint"`
	Orientation CapabilityState       `json:"orientation"`
}

func DetectCapabilities(snapshot Snapshot) Capabilities {
	return Capabilities{
		Thunderbolt: detectThunderboltCapability(snapshot),
		CardReader:  detectCardReaderCapability(snapshot),
		Fingerprint: detectFingerprintCapability(snapshot),
		Orientation: detectOrientationCapability(snapshot),
	}
}

func detectThunderboltCapability(snapshot Snapshot) CapabilityState {
	if snapshot.ThunderboltHost {
		return CapabilityState{
			Present: true,
			Details: "Thunderbolt host/domain present",
		}
	}

	if len(snapshot.Thunderbolt) != 0 {
		return CapabilityState{
			Present: true,
			Details: "Thunderbolt controller/domain present",
		}
	}

	for _, device := range snapshot.PCIDevices {
		if strings.EqualFold(device.Driver, "thunderbolt") {
			return CapabilityState{
				Present: true,
				Details: "Thunderbolt controller present",
			}
		}
	}

	for _, line := range snapshot.PCI {
		if strings.Contains(strings.ToLower(line), "thunderbolt") {
			return CapabilityState{
				Present: true,
				Details: "Thunderbolt controller present",
			}
		}
	}

	return CapabilityState{}
}

func detectCardReaderCapability(snapshot Snapshot) CapabilityState {
	for _, line := range snapshot.PCI {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "card reader") ||
			strings.Contains(lower, "cardreader") {
			return CapabilityState{
				Present: true,
				Details: "card-reader controller present",
			}
		}
	}

	for _, device := range snapshot.PCIDevices {
		driver := strings.ToLower(device.Driver)
		if strings.Contains(driver, "rtsx") ||
			strings.Contains(driver, "sdhci") {
			return CapabilityState{
				Present: true,
				Details: "media-reader controller present",
			}
		}
	}

	for _, device := range snapshot.Block {
		if device.Removable {
			return CapabilityState{
				Present: true,
				Details: "removable media device present",
			}
		}
	}

	return CapabilityState{}
}

func detectFingerprintCapability(snapshot Snapshot) FingerprintCapability {
	upstream := len(snapshot.Fingerprint.Devices) != 0
	if upstream {
		return FingerprintCapability{
			Present:           true,
			UpstreamSupported: true,
		}
	}

	for _, line := range snapshot.USB {
		lower := strings.ToLower(line)
		if strings.Contains(lower, "fingerprint") ||
			strings.Contains(lower, "biometric") {
			return FingerprintCapability{
				Present: true,
			}
		}
	}

	return FingerprintCapability{}
}

func detectOrientationCapability(snapshot Snapshot) CapabilityState {
	for _, device := range snapshot.IIO {
		if orientation.IsSensor(device.Name, device.Channels) {
			return CapabilityState{
				Present: true,
				Details: "orientation-capable IIO sensor present",
			}
		}
	}

	return CapabilityState{}
}
