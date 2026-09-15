package laptop13amd7040

import (
	"fmt"
	"strings"

	"github.com/bakanura/gjallarOS/internal/hardware/deviceprobe"
)

type Status string

const (
	StatusPass Status = "PASS"
	StatusWarn Status = "WARN"
	StatusFail Status = "FAIL"
)

type Result struct {
	Status Status
	Gate   string
	Detail string
}

type Report struct {
	Results []Result
}

func Evaluate(snapshot deviceprobe.Snapshot) Report {
	return Report{Results: []Result{
		identity(snapshot),
		graphics(snapshot),
		wifi(snapshot),
		bluetooth(snapshot),
		fingerprint(snapshot),
		audio(snapshot),
		touchpad(snapshot),
		sensors(snapshot),
		battery(snapshot),
		chargeControl(snapshot),
		backlight(snapshot),
		camera(snapshot),
		embeddedController(snapshot),
		fan(snapshot),
		amdPState(snapshot),
		platformProfile(snapshot),
		thunderbolt(snapshot),
		tpm(snapshot),
	}}
}

func MatchesDevice(snapshot deviceprobe.Snapshot) bool {
	return equal(snapshot.SysVendor, "Framework") &&
		equal(snapshot.ProductName, "Laptop 13 (AMD Ryzen 7040Series)") &&
		equal(snapshot.BoardName, "FRANMDCP07")
}

func identity(snapshot deviceprobe.Snapshot) Result {
	if MatchesDevice(snapshot) {
		return pass("identity", "Framework Laptop 13 AMD Ryzen 7040 / FRANMDCP07 matched")
	}
	return fail("identity", fmt.Sprintf(
		"unexpected device identity: vendor=%q product=%q board=%q",
		snapshot.SysVendor, snapshot.ProductName, snapshot.BoardName,
	))
}

func graphics(snapshot deviceprobe.Snapshot) Result {
	for _, d := range snapshot.PCIDevices {
		if equal(d.Vendor, "0x1002") && equal(d.Device, "0x15bf") {
			if equal(d.Driver, "amdgpu") {
				return pass("graphics", "Radeon 780M 1002:15bf bound to amdgpu")
			}
			return fail("graphics", fmt.Sprintf("Radeon 780M present but driver=%q", d.Driver))
		}
	}
	return fail("graphics", "Radeon 780M 1002:15bf not observed")
}

func wifi(snapshot deviceprobe.Snapshot) Result {
	for _, d := range snapshot.PCIDevices {
		if equal(d.Vendor, "0x10ec") && equal(d.Device, "0xb852") {
			if equal(d.Driver, "rtw89_8852be") {
				return pass("wifi", "RTL8852BE bound to rtw89_8852be")
			}
			return fail("wifi", fmt.Sprintf("RTL8852BE present but driver=%q", d.Driver))
		}
	}
	return warn("wifi", "RTL8852BE 10ec:b852 not observed")
}

func bluetooth(snapshot deviceprobe.Snapshot) Result {
	for _, d := range snapshot.USBDevices {
		if equal(d.Vendor, "13d3") && equal(d.Product, "3571") {
			return pass("bluetooth", "Realtek Bluetooth radio 13d3:3571 observed")
		}
	}
	return warn("bluetooth", "Realtek Bluetooth radio 13d3:3571 not observed")
}

func fingerprint(snapshot deviceprobe.Snapshot) Result {
	for _, d := range snapshot.USBDevices {
		if equal(d.Vendor, "27c6") && equal(d.Product, "609c") {
			if len(snapshot.Fingerprint.Devices) != 0 {
				return pass("fingerprint", "Goodix 27c6:609c exposed through fprintd")
			}
			return warn("fingerprint", "Goodix 27c6:609c present but not exposed through fprintd")
		}
	}
	return warn("fingerprint", "Goodix fingerprint reader not observed")
}

func audio(snapshot deviceprobe.Snapshot) Result {
	if len(snapshot.Audio) >= 2 {
		return pass("audio", fmt.Sprintf("%d ALSA audio cards observed", len(snapshot.Audio)))
	}
	if len(snapshot.Audio) != 0 {
		return warn("audio", fmt.Sprintf("only %d ALSA audio card observed", len(snapshot.Audio)))
	}
	return fail("audio", "no ALSA audio cards observed")
}

func touchpad(snapshot deviceprobe.Snapshot) Result {
	for _, d := range snapshot.HIDDevices {
		if equal(d.Vendor, "093a") && equal(d.Product, "0274") &&
			equal(d.Driver, "hid-multitouch") {
			return pass("touchpad", "PixArt 093a:0274 bound to hid-multitouch")
		}
	}
	return fail("touchpad", "Framework PixArt touchpad 093a:0274 not observed")
}

func sensors(snapshot deviceprobe.Snapshot) Result {
	if len(snapshot.IIO) != 0 {
		return pass("sensors", fmt.Sprintf("%d IIO sensor device(s) observed", len(snapshot.IIO)))
	}
	return warn("sensors", "no IIO sensor devices observed")
}

func battery(snapshot deviceprobe.Snapshot) Result {
	for _, supply := range snapshot.PowerSupplies {
		if equal(supply.Type, "Battery") {
			detail := fmt.Sprintf("%s battery observed", supply.Name)
			if supply.Capacity != "" {
				detail += fmt.Sprintf(" at %s%%", supply.Capacity)
			}
			if supply.Status != "" {
				detail += fmt.Sprintf(" (%s)", supply.Status)
			}
			return pass("battery", detail)
		}
	}
	return fail("battery", "no battery power supply observed")
}

func chargeControl(snapshot deviceprobe.Snapshot) Result {
	for _, supply := range snapshot.PowerSupplies {
		if !equal(supply.Type, "Battery") {
			continue
		}

		if supply.ChargeControlStartThreshold != "" ||
			supply.ChargeControlEndThreshold != "" ||
			supply.ChargeStartThreshold != "" ||
			supply.ChargeStopThreshold != "" {
			return pass("charge-control", fmt.Sprintf(
				"%s exposes battery charge threshold controls",
				supply.Name,
			))
		}
	}
	return warn("charge-control", "battery present but no sysfs charge threshold controls exposed")
}

func backlight(snapshot deviceprobe.Snapshot) Result {
	for _, device := range snapshot.Backlights {
		if equal(device.Name, "amdgpu_bl1") {
			return pass("backlight", fmt.Sprintf(
				"%s active brightness=%s actual=%s max=%s",
				device.Name,
				device.Brightness,
				device.ActualBrightness,
				device.MaxBrightness,
			))
		}
	}
	if len(snapshot.Backlights) != 0 {
		return warn("backlight", fmt.Sprintf(
			"%d backlight device(s) observed but amdgpu_bl1 missing",
			len(snapshot.Backlights),
		))
	}
	return fail("backlight", "no backlight device observed")
}

func camera(snapshot deviceprobe.Snapshot) Result {
	if len(snapshot.VideoDevices) == 0 {
		return warn("camera", "no V4L2 camera devices observed")
	}

	return pass("camera", fmt.Sprintf(
		"%d V4L2 video device(s) observed",
		len(snapshot.VideoDevices),
	))
}

func embeddedController(snapshot deviceprobe.Snapshot) Result {
	if snapshot.EmbeddedControl.Present &&
		equal(snapshot.EmbeddedControl.Name, "cros_ec") {
		return pass("ec", "ChromeOS EC interface cros_ec observed")
	}
	return fail("ec", "Framework cros_ec interface not observed")
}

func fan(snapshot deviceprobe.Snapshot) Result {
	for _, device := range snapshot.Fans {
		if !equal(device.Name, "cros_ec") {
			continue
		}

		detail := fmt.Sprintf("cros_ec fan telemetry input=%s RPM", device.Input)
		if device.Target != "" {
			detail += fmt.Sprintf(" target=%s RPM", device.Target)
		}
		return pass("fan", detail)
	}
	return warn("fan", "cros_ec present but no fan telemetry observed")
}

func amdPState(snapshot deviceprobe.Snapshot) Result {
	if equal(snapshot.AMDPower.PStateStatus, "active") {
		return pass("amd-pstate", "amd_pstate is active")
	}
	if snapshot.AMDPower.PStateStatus == "" {
		return fail("amd-pstate", "amd_pstate status unavailable")
	}
	return fail("amd-pstate", fmt.Sprintf(
		"amd_pstate status=%q, expected active",
		snapshot.AMDPower.PStateStatus,
	))
}

func platformProfile(snapshot deviceprobe.Snapshot) Result {
	if snapshot.AMDPower.PlatformProfile == "" {
		return warn("platform-profile", "ACPI platform profile unavailable")
	}

	expected := map[string]bool{
		"low-power":   false,
		"balanced":    false,
		"performance": false,
	}
	for _, choice := range snapshot.AMDPower.PlatformProfileChoices {
		if _, ok := expected[choice]; ok {
			expected[choice] = true
		}
	}

	if expected["low-power"] && expected["balanced"] && expected["performance"] {
		return pass("platform-profile", fmt.Sprintf(
			"ACPI platform profile=%s choices=low-power,balanced,performance",
			snapshot.AMDPower.PlatformProfile,
		))
	}

	return warn("platform-profile", fmt.Sprintf(
		"ACPI platform profile=%s with incomplete choices %v",
		snapshot.AMDPower.PlatformProfile,
		snapshot.AMDPower.PlatformProfileChoices,
	))
}

func thunderbolt(snapshot deviceprobe.Snapshot) Result {
	if snapshot.ThunderboltHost && len(snapshot.Thunderbolt) >= 2 {
		return pass("usb4", fmt.Sprintf("%d Thunderbolt/USB4 domains observed", len(snapshot.Thunderbolt)))
	}
	if snapshot.ThunderboltHost {
		return warn("usb4", "Thunderbolt/USB4 host present but expected domains incomplete")
	}
	return fail("usb4", "Thunderbolt/USB4 host not observed")
}

func tpm(snapshot deviceprobe.Snapshot) Result {
	for _, d := range snapshot.PCIDevices {
		if equal(d.Vendor, "0x1022") && equal(d.Device, "0x15c7") && equal(d.Driver, "ccp") {
			return pass("platform-security", "Phoenix CCP/PSP 1022:15c7 bound to ccp")
		}
	}
	return warn("platform-security", "Phoenix CCP/PSP not observed")
}

func pass(gate, detail string) Result { return Result{StatusPass, gate, detail} }
func warn(gate, detail string) Result { return Result{StatusWarn, gate, detail} }
func fail(gate, detail string) Result { return Result{StatusFail, gate, detail} }

func equal(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}
