package usbtrust

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

const SchemaVersion = 1

type DeviceClass string

const (
	ClassInternal DeviceClass = "internal"
	ClassExternal DeviceClass = "external"
)

type IdentityStrength string

const (
	StrengthDescriptor               IdentityStrength = "descriptor"
	StrengthSerialDescriptor         IdentityStrength = "serial+descriptor"
	StrengthSerialDescriptorTopology IdentityStrength = "serial+descriptor+topology"
	StrengthHardwareAttested         IdentityStrength = "hardware-attested"
)

type Identity struct {
	VIDPID      string   `json:"vidPid"`
	Serial      string   `json:"serial,omitempty"`
	Name        string   `json:"name,omitempty"`
	Hash        string   `json:"hash"`
	ParentHash  string   `json:"parentHash,omitempty"`
	Interfaces  []string `json:"interfaces,omitempty"`
	ConnectType string   `json:"connectType,omitempty"`
	Port        string   `json:"port,omitempty"`
}

type Attestation struct {
	Type        string `json:"type"`
	Identity    string `json:"identity"`
	Fingerprint string `json:"fingerprint"`
}

type Device struct {
	ID             string           `json:"id"`
	Role           string           `json:"role,omitempty"`
	Class          DeviceClass      `json:"class"`
	Portable       bool             `json:"portable"`
	ExpectedByODDC bool             `json:"expectedByOddc"`
	Strength       IdentityStrength `json:"identityStrength"`
	Identity       Identity         `json:"identity"`
	Attestation    *Attestation     `json:"attestation,omitempty"`
	FirstAccepted  string           `json:"firstAccepted"`
	LastAccepted   string           `json:"lastAccepted"`
}

type Document struct {
	Schema    int      `json:"schema"`
	MachineID string   `json:"machineId"`
	ODDCModel string   `json:"oddcModel"`
	Revision  uint64   `json:"revision"`
	Devices   []Device `json:"devices"`
}

func (d *Document) Normalize() {
	for i := range d.Devices {
		sort.Strings(d.Devices[i].Identity.Interfaces)
	}

	sort.Slice(d.Devices, func(i, j int) bool {
		return d.Devices[i].ID < d.Devices[j].ID
	})
}

func (d Document) Validate() error {
	if d.Schema != SchemaVersion {
		return fmt.Errorf("unsupported USB trust schema %d", d.Schema)
	}

	if strings.TrimSpace(d.MachineID) == "" {
		return errors.New("machineId is required")
	}

	if strings.TrimSpace(d.ODDCModel) == "" {
		return errors.New("oddcModel is required")
	}

	if d.Revision == 0 {
		return errors.New("revision must be greater than zero")
	}

	seen := make(map[string]struct{}, len(d.Devices))

	for _, device := range d.Devices {
		if strings.TrimSpace(device.ID) == "" {
			return errors.New("device id is required")
		}

		if _, exists := seen[device.ID]; exists {
			return fmt.Errorf("duplicate USB trust device id %q", device.ID)
		}
		seen[device.ID] = struct{}{}

		switch device.Class {
		case ClassInternal, ClassExternal:
		default:
			return fmt.Errorf(
				"device %q has invalid class %q",
				device.ID,
				device.Class,
			)
		}

		if strings.TrimSpace(device.Identity.VIDPID) == "" {
			return fmt.Errorf("device %q has no VID:PID", device.ID)
		}

		if strings.TrimSpace(device.Identity.Hash) == "" {
			return fmt.Errorf("device %q has no descriptor hash", device.ID)
		}

		if device.Class == ClassInternal && !device.ExpectedByODDC {
			return fmt.Errorf(
				"internal device %q is not backed by ODDC expectation",
				device.ID,
			)
		}

		if device.Class == ClassInternal && device.Portable {
			return fmt.Errorf(
				"internal device %q cannot be portable",
				device.ID,
			)
		}
	}

	return nil
}

func Canonical(d Document) ([]byte, error) {
	d.Normalize()

	if err := d.Validate(); err != nil {
		return nil, err
	}

	return json.Marshal(d)
}
