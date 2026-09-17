package usbtrust

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

type RiskSeverity string

const (
	RiskInfo   RiskSeverity = "info"
	RiskReview RiskSeverity = "review"
	RiskHigh   RiskSeverity = "high"
)

const (
	RiskHIDInput            = "hid-input"
	RiskMassStorage         = "mass-storage"
	RiskNetwork             = "network"
	RiskWirelessController  = "wireless-controller"
	RiskVendorSpecific      = "vendor-specific"
	RiskHub                 = "hub"
	RiskBillboard           = "billboard"
	RiskComposite           = "composite"
	RiskHIDStorageComposite = "hid-storage-composite"
	RiskHIDNetworkComposite = "hid-network-composite"
)

type RiskFinding struct {
	Code     string       `json:"code"`
	Severity RiskSeverity `json:"severity"`
	Detail   string       `json:"detail"`
}

var usbInterfacePattern = regexp.MustCompile(
	`^[0-9a-fA-F]{2}:[0-9a-fA-F]{2}:[0-9a-fA-F]{2}$`,
)

// AssessIdentityRisk classifies capabilities exposed by one observed USB
// identity. It intentionally does not identify products or vendors.
//
// These are capability indicators, not proof of malicious intent.
// Trust decisions remain the responsibility of Audit and the privileged
// USB trust owner.
func AssessIdentityRisk(
	identity Identity,
) ([]RiskFinding, error) {
	interfaces := map[string]struct{}{}
	classes := map[string]struct{}{}

	for _, raw := range identity.Interfaces {
		iface := strings.ToLower(strings.TrimSpace(raw))
		if iface == "" {
			continue
		}

		if !usbInterfacePattern.MatchString(iface) {
			return nil, fmt.Errorf(
				"invalid USB interface descriptor %q",
				raw,
			)
		}

		interfaces[iface] = struct{}{}
		classes[iface[:2]] = struct{}{}
	}

	var findings []RiskFinding

	add := func(
		code string,
		severity RiskSeverity,
		detail string,
	) {
		findings = append(findings, RiskFinding{
			Code:     code,
			Severity: severity,
			Detail:   detail,
		})
	}

	if hasClass(classes, "03") {
		add(
			RiskHIDInput,
			RiskHigh,
			"exposes USB HID input capability",
		)
	}

	if hasClass(classes, "08") {
		add(
			RiskMassStorage,
			RiskHigh,
			"exposes USB mass-storage capability",
		)
	}

	if hasClass(classes, "02") || hasClass(classes, "0a") {
		add(
			RiskNetwork,
			RiskHigh,
			"exposes USB communications/network capability",
		)
	}

	if hasClass(classes, "e0") {
		add(
			RiskWirelessController,
			RiskReview,
			"exposes a USB wireless-controller interface",
		)
	}

	if hasClass(classes, "ff") {
		add(
			RiskVendorSpecific,
			RiskReview,
			"exposes an opaque vendor-specific USB interface",
		)
	}

	if hasClass(classes, "09") {
		add(
			RiskHub,
			RiskInfo,
			"exposes a USB hub interface",
		)
	}

	if hasClass(classes, "11") {
		add(
			RiskBillboard,
			RiskInfo,
			"exposes a USB Billboard interface",
		)
	}

	if len(interfaces) > 1 {
		add(
			RiskComposite,
			RiskReview,
			"exposes multiple distinct USB interfaces",
		)
	}

	if hasClass(classes, "03") && hasClass(classes, "08") {
		add(
			RiskHIDStorageComposite,
			RiskHigh,
			"combines HID input and mass-storage capabilities",
		)
	}

	if hasClass(classes, "03") &&
		(hasClass(classes, "02") || hasClass(classes, "0a")) {
		add(
			RiskHIDNetworkComposite,
			RiskHigh,
			"combines HID input and network capabilities",
		)
	}

	sort.Slice(findings, func(i, j int) bool {
		return findings[i].Code < findings[j].Code
	})

	return findings, nil
}

func hasClass(
	classes map[string]struct{},
	class string,
) bool {
	_, exists := classes[class]
	return exists
}
