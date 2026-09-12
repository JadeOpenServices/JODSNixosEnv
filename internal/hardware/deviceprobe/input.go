package deviceprobe

import (
	"github.com/bakanura/gjallarOS/internal/hardware/inputclass"
	"path/filepath"
	"sort"
	"strings"
)

type InputCapabilities struct {
	EV  string `json:"ev,omitempty"`
	Key string `json:"key,omitempty"`
	Abs string `json:"abs,omitempty"`
	Rel string `json:"rel,omitempty"`
	MSC string `json:"msc,omitempty"`
	LED string `json:"led,omitempty"`
	SW  string `json:"sw,omitempty"`
}

type InputClassification struct {
	Touchscreen bool `json:"touchscreen"`
	Pen         bool `json:"pen"`
	Keyboard    bool `json:"keyboard"`
	Buttons     bool `json:"buttons"`
}

type InputDevice struct {
	Event          string              `json:"event"`
	Name           string              `json:"name"`
	Driver         string              `json:"driver,omitempty"`
	Properties     string              `json:"properties,omitempty"`
	Capabilities   InputCapabilities   `json:"capabilities"`
	Classification InputClassification `json:"classification"`
}

func inputDevices(sysRoot string) []InputDevice {
	paths, _ := filepath.Glob(filepath.Join(sysRoot, "class/input/event*"))
	devices := make([]InputDevice, 0, len(paths))

	for _, path := range paths {
		name := read(filepath.Join(path, "device/name"))
		if name == "" {
			continue
		}

		caps := filepath.Join(path, "device/capabilities")
		properties := read(filepath.Join(path, "device/properties"))
		capabilities := InputCapabilities{
			EV:  read(filepath.Join(caps, "ev")),
			Key: read(filepath.Join(caps, "key")),
			Abs: read(filepath.Join(caps, "abs")),
			Rel: read(filepath.Join(caps, "rel")),
			MSC: read(filepath.Join(caps, "msc")),
			LED: read(filepath.Join(caps, "led")),
			SW:  read(filepath.Join(caps, "sw")),
		}

		classificationDevice := inputclass.Device{
			Name:       name,
			Properties: properties,
			Key:        capabilities.Key,
			Abs:        capabilities.Abs,
		}

		devices = append(devices, InputDevice{
			Event:        filepath.Base(path),
			Name:         name,
			Driver:       driverName(filepath.Join(path, "device/driver")),
			Properties:   properties,
			Capabilities: capabilities,
			Classification: InputClassification{
				Touchscreen: inputclass.IsTouchscreen(classificationDevice),
				Pen:         inputclass.IsPen(classificationDevice),
				Keyboard:    inputclass.IsKeyboard(classificationDevice),
				Buttons:     inputclass.IsButtonInput(classificationDevice),
			},
		})
	}

	sort.Slice(devices, func(i, j int) bool {
		return devices[i].Event < devices[j].Event
	})

	return devices
}

type IIODevice struct {
	Device   string   `json:"device"`
	Name     string   `json:"name"`
	Driver   string   `json:"driver,omitempty"`
	Channels []string `json:"channels,omitempty"`
}

func iioDevices(sysRoot string) []IIODevice {
	paths, _ := filepath.Glob(
		filepath.Join(sysRoot, "bus/iio/devices/iio:device*"),
	)
	devices := make([]IIODevice, 0, len(paths))

	for _, path := range paths {
		name := read(filepath.Join(path, "name"))
		if name == "" {
			continue
		}

		devices = append(devices, IIODevice{
			Device:   filepath.Base(path),
			Name:     name,
			Driver:   driverName(filepath.Join(path, "driver")),
			Channels: iioChannels(path),
		})
	}

	sort.Slice(devices, func(i, j int) bool {
		return devices[i].Device < devices[j].Device
	})

	return devices
}

func iioChannels(path string) []string {
	entries, _ := filepath.Glob(filepath.Join(path, "in_*"))
	channels := make([]string, 0, len(entries))

	for _, entry := range entries {
		name := filepath.Base(entry)

		if strings.HasSuffix(name, "_raw") ||
			strings.HasSuffix(name, "_scale") ||
			strings.HasSuffix(name, "_offset") {
			channels = append(channels, name)
		}
	}

	sort.Strings(channels)
	return channels
}
