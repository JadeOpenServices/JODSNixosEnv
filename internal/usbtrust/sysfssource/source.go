package sysfssource

import (
	"regexp"
	"strings"

	"github.com/bakanura/gjallarOS/internal/hardware/deviceprobe"
	"github.com/bakanura/gjallarOS/internal/usbtrust"
)

var rootHubPath = regexp.MustCompile(`^usb[0-9]+$`)

// Observed converts the kernel/sysfs USB inventory into minimal USB trust
// observations. It preserves only facts actually exposed by this inventory and
// deliberately does not invent descriptor hashes, topology identity,
// interfaces, serials, or connection classification.
func Observed(snapshot deviceprobe.Snapshot) []usbtrust.ObservedDevice {
	out := make([]usbtrust.ObservedDevice, 0, len(snapshot.USBDevices))

	for _, device := range snapshot.USBDevices {
		if rootHubPath.MatchString(device.Path) {
			continue
		}

		vendor := strings.ToLower(strings.TrimSpace(device.Vendor))
		product := strings.ToLower(strings.TrimSpace(device.Product))

		if vendor == "" || product == "" {
			continue
		}

		out = append(out, usbtrust.ObservedDevice{
			RuntimeID: device.Path,
			Identity: usbtrust.Identity{
				VIDPID: vendor + ":" + product,
			},
		})
	}

	return out
}
