package deviceprobe

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

type FingerprintDevice struct {
	Path string `json:"path"`
	Name string `json:"name,omitempty"`
}

type FingerprintState struct {
	Available bool                `json:"available"`
	Devices   []FingerprintDevice `json:"devices"`
	Warning   string              `json:"warning,omitempty"`
}

func fingerprintState(ctx context.Context) FingerprintState {
	state := FingerprintState{
		Devices: []FingerprintDevice{},
	}

	out, err := command(
		ctx,
		"busctl",
		"--system",
		"call",
		"net.reactivated.Fprint",
		"/net/reactivated/Fprint/Manager",
		"net.reactivated.Fprint.Manager",
		"GetDevices",
	)
	if err != nil {
		state.Warning = err.Error()
		return state
	}

	paths, err := parseObjectPaths(out)
	if err != nil {
		state.Warning = err.Error()
		return state
	}

	state.Available = true

	for _, path := range paths {
		device := FingerprintDevice{Path: path}

		name, err := command(
			ctx,
			"busctl",
			"--system",
			"get-property",
			"net.reactivated.Fprint",
			path,
			"net.reactivated.Fprint.Device",
			"name",
		)
		if err == nil {
			device.Name = parseBusctlString(name)
		}

		state.Devices = append(state.Devices, device)
	}

	return state
}

func parseObjectPaths(value string) ([]string, error) {
	fields := strings.Fields(value)
	if len(fields) < 2 || fields[0] != "ao" {
		return nil, fmt.Errorf("unexpected fprintd GetDevices response")
	}

	count, err := strconv.Atoi(fields[1])
	if err != nil || count < 0 {
		return nil, fmt.Errorf("invalid fprintd device count")
	}
	if len(fields) != count+2 {
		return nil, fmt.Errorf("incomplete fprintd device response")
	}

	out := make([]string, 0, count)
	for _, field := range fields[2:] {
		path := strings.Trim(field, `"`)
		if !strings.HasPrefix(path, "/net/reactivated/Fprint/Device/") {
			return nil, fmt.Errorf("unexpected fprintd device path")
		}
		out = append(out, path)
	}

	return out, nil
}

func parseBusctlString(value string) string {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, `s "`) || !strings.HasSuffix(value, `"`) {
		return ""
	}
	return strings.TrimSuffix(strings.TrimPrefix(value, `s "`), `"`)
}
