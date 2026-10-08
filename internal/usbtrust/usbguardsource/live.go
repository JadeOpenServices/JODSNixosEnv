package usbguardsource

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/JadeOpenServices/gjallarOS/internal/usbtrust"
)

var rootUSBPortPattern = regexp.MustCompile(`^usb[0-9]+$`)

type Runner interface {
	Output(
		ctx context.Context,
		name string,
		args ...string,
	) ([]byte, error)
}

type ExecRunner struct{}

func (ExecRunner) Output(
	ctx context.Context,
	name string,
	args ...string,
) ([]byte, error) {
	output, err := exec.CommandContext(
		ctx,
		name,
		args...,
	).CombinedOutput()

	if err != nil {
		return nil, fmt.Errorf(
			"%s %s: %w: %s",
			name,
			strings.Join(args, " "),
			err,
			strings.TrimSpace(string(output)),
		)
	}

	return output, nil
}

type LiveSource struct {
	Runner Runner
	Binary string
}

// Observed returns enforcement-relevant USB devices from the running
// USBGuard daemon.
//
// USB host-controller root hubs are infrastructure rather than
// authorizable peripheral identities, so they are filtered by generic
// topology/interface shape rather than vendor or product identifiers.
func (s LiveSource) Observed(
	ctx context.Context,
) ([]usbtrust.ObservedDevice, error) {
	if s.Runner == nil {
		return nil, fmt.Errorf(
			"USBGuard observation runner is unavailable",
		)
	}

	binary := strings.TrimSpace(s.Binary)
	if binary == "" {
		binary = "usbguard"
	}

	output, err := s.Runner.Output(
		ctx,
		binary,
		"list-devices",
	)
	if err != nil {
		return nil, fmt.Errorf(
			"list USBGuard devices: %w",
			err,
		)
	}

	parsed, err := ParseLines(string(output))
	if err != nil {
		return nil, fmt.Errorf(
			"parse USBGuard live devices: %w",
			err,
		)
	}

	devices := make(
		[]usbtrust.ObservedDevice,
		0,
		len(parsed),
	)

	for _, observation := range parsed {
		if strings.TrimSpace(
			observation.Device.RuntimeID,
		) == "" {
			return nil, fmt.Errorf(
				"USBGuard live observation %s has no runtime id",
				observation.Device.Identity.VIDPID,
			)
		}

		if controllerInfrastructure(
			observation.Device.Identity,
		) {
			continue
		}

		devices = append(
			devices,
			observation.Device,
		)
	}

	return devices, nil
}

func controllerInfrastructure(
	identity usbtrust.Identity,
) bool {
	if !rootUSBPortPattern.MatchString(identity.Port) {
		return false
	}

	if identity.ConnectType != "" {
		return false
	}

	if len(identity.Interfaces) == 0 {
		return false
	}

	for _, iface := range identity.Interfaces {
		if len(iface) < 2 ||
			!strings.EqualFold(iface[:2], "09") {
			return false
		}
	}

	return true
}
