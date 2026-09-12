package zbookx2g4

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
	Status Status `json:"status"`
	Gate   string `json:"gate"`
	Detail string `json:"detail"`
}

type Report struct {
	Results []Result `json:"results"`
}

func Evaluate(snapshot deviceprobe.Snapshot) Report {
	results := []Result{
		identity(snapshot),
		graphics(snapshot),
		touch(snapshot),
		pen(snapshot),
		keyboard(snapshot),
		quickKeys(snapshot),
		sensors(snapshot),
		fingerprint(snapshot),
		thunderbolt(snapshot),
		audio(snapshot),
		cardReader(snapshot),
	}

	return Report{Results: results}
}

func MatchesDevice(snapshot deviceprobe.Snapshot) bool {
	return equal(snapshot.SysVendor, "HP") &&
		equal(snapshot.ProductName, "HP ZBook x2 G4") &&
		equal(snapshot.BoardName, "824C")
}

func identity(snapshot deviceprobe.Snapshot) Result {
	if MatchesDevice(snapshot) {
		return Result{
			Status: StatusPass,
			Gate:   "identity",
			Detail: "HP ZBook x2 G4 / board 824C matched",
		}
	}

	return Result{
		Status: StatusFail,
		Gate:   "identity",
		Detail: fmt.Sprintf(
			"unexpected device identity: vendor=%q product=%q board=%q",
			snapshot.SysVendor,
			snapshot.ProductName,
			snapshot.BoardName,
		),
	}
}

func graphics(snapshot deviceprobe.Snapshot) Result {
	for _, device := range snapshot.PCIDevices {
		if equal(device.Vendor, "0x10de") &&
			equal(device.Device, "0x13b4") {
			return Result{
				Status: StatusPass,
				Gate:   "graphics",
				Detail: fmt.Sprintf(
					"Quadro M620 detected at %s",
					device.Address,
				),
			}
		}
	}

	return Result{
		Status: StatusWarn,
		Gate:   "graphics",
		Detail: "Quadro M620 PCI ID 10de:13b4 not observed",
	}
}

func touch(snapshot deviceprobe.Snapshot) Result {
	if !hasHID(snapshot, "03eb", "8abb") {
		return Result{
			Status: StatusWarn,
			Gate:   "touch",
			Detail: "ZBook touchscreen HID 03eb:8abb not observed",
		}
	}

	for _, device := range snapshot.Input {
		if device.Classification.Touchscreen {
			return Result{
				Status: StatusPass,
				Gate:   "touch",
				Detail: fmt.Sprintf(
					"ZBook touchscreen HID 03eb:8abb present; touchscreen input observed on %s (%s)",
					device.Event,
					device.Name,
				),
			}
		}
	}

	return Result{
		Status: StatusWarn,
		Gate:   "touch",
		Detail: "ZBook touchscreen HID 03eb:8abb present but no touchscreen input classified",
	}
}

func pen(snapshot deviceprobe.Snapshot) Result {
	if !hasHID(snapshot, "056a", "016c") {
		return Result{
			Status: StatusWarn,
			Gate:   "pen",
			Detail: "ZBook Wacom digitizer HID 056a:016c not observed",
		}
	}

	for _, device := range snapshot.Input {
		if device.Classification.Pen {
			return Result{
				Status: StatusPass,
				Gate:   "pen",
				Detail: fmt.Sprintf(
					"ZBook Wacom digitizer HID 056a:016c present; pen input observed on %s (%s)",
					device.Event,
					device.Name,
				),
			}
		}
	}

	return Result{
		Status: StatusWarn,
		Gate:   "pen",
		Detail: "ZBook Wacom digitizer HID 056a:016c present but no pen input classified",
	}
}

func hasHID(snapshot deviceprobe.Snapshot, vendor, product string) bool {
	for _, device := range snapshot.HIDDevices {
		if equal(device.Vendor, vendor) &&
			equal(device.Product, product) {
			return true
		}
	}

	return false
}

func keyboard(snapshot deviceprobe.Snapshot) Result {
	for _, device := range snapshot.Input {
		if device.Classification.Keyboard {
			return Result{
				Status: StatusPass,
				Gate:   "keyboard",
				Detail: fmt.Sprintf(
					"keyboard-like input observed on %s (%s)",
					device.Event,
					device.Name,
				),
			}
		}
	}

	return Result{
		Status: StatusWarn,
		Gate:   "keyboard",
		Detail: "no keyboard-like input classified",
	}
}

func quickKeys(snapshot deviceprobe.Snapshot) Result {
	for _, device := range snapshot.HIDDevices {
		if equal(device.Vendor, QuickKeysVendorID) &&
			equal(device.Product, QuickKeysProductID) {
			return Result{
				Status: StatusPass,
				Gate:   "quick-keys",
				Detail: fmt.Sprintf(
					"HP Quick Keys %s:%s observed as %s using driver %q with hidraw=%v",
					device.Vendor,
					device.Product,
					device.Name,
					device.Driver,
					device.Hidraw,
				),
			}
		}
	}

	return Result{
		Status: StatusWarn,
		Gate:   "quick-keys",
		Detail: "HP Quick Keys HID 03f0:0a56 not observed",
	}
}

func thunderbolt(snapshot deviceprobe.Snapshot) Result {
	capabilities := deviceprobe.DetectCapabilities(snapshot)

	if !capabilities.Thunderbolt.Present {
		return Result{
			Status: StatusWarn,
			Gate:   "thunderbolt",
			Detail: "Thunderbolt capability not observed",
		}
	}

	detail := capabilities.Thunderbolt.Details
	if detail == "" {
		detail = "Thunderbolt capability present"
	}

	return Result{
		Status: StatusPass,
		Gate:   "thunderbolt",
		Detail: detail,
	}
}

func audio(snapshot deviceprobe.Snapshot) Result {
	if len(snapshot.Audio) == 0 {
		return Result{
			Status: StatusWarn,
			Gate:   "audio",
			Detail: "no ALSA audio cards observed",
		}
	}

	return Result{
		Status: StatusPass,
		Gate:   "audio",
		Detail: fmt.Sprintf(
			"%d ALSA audio card(s) observed",
			len(snapshot.Audio),
		),
	}
}

func cardReader(snapshot deviceprobe.Snapshot) Result {
	capabilities := deviceprobe.DetectCapabilities(snapshot)

	if !capabilities.CardReader.Present {
		return Result{
			Status: StatusWarn,
			Gate:   "card-reader",
			Detail: "card-reader capability not observed",
		}
	}

	detail := capabilities.CardReader.Details
	if detail == "" {
		detail = "card-reader capability present"
	}

	return Result{
		Status: StatusPass,
		Gate:   "card-reader",
		Detail: detail,
	}
}

func fingerprint(snapshot deviceprobe.Snapshot) Result {
	capabilities := deviceprobe.DetectCapabilities(snapshot)

	if !capabilities.Fingerprint.Present {
		return Result{
			Status: StatusWarn,
			Gate:   "fingerprint",
			Detail: "fingerprint hardware not observed",
		}
	}

	if !capabilities.Fingerprint.UpstreamSupported {
		return Result{
			Status: StatusWarn,
			Gate:   "fingerprint",
			Detail: "fingerprint hardware present; upstream fprintd/libfprint does not expose it",
		}
	}

	names := make([]string, 0, len(snapshot.Fingerprint.Devices))
	for _, device := range snapshot.Fingerprint.Devices {
		if device.Name != "" {
			names = append(names, device.Name)
		}
	}

	detail := fmt.Sprintf(
		"%d fingerprint reader(s) visible through upstream fprintd/libfprint",
		len(snapshot.Fingerprint.Devices),
	)
	if len(names) > 0 {
		detail += ": " + strings.Join(names, ", ")
	}

	return Result{
		Status: StatusPass,
		Gate:   "fingerprint",
		Detail: detail,
	}
}

func sensors(snapshot deviceprobe.Snapshot) Result {
	if len(snapshot.IIO) == 0 {
		return Result{
			Status: StatusWarn,
			Gate:   "sensors",
			Detail: "no IIO sensor devices observed",
		}
	}

	return Result{
		Status: StatusPass,
		Gate:   "sensors",
		Detail: fmt.Sprintf(
			"%d IIO sensor devices observed",
			len(snapshot.IIO),
		),
	}
}

func equal(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), strings.TrimSpace(b))
}
