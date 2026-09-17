package oddcsource

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bakanura/gjallarOS/internal/usbtrust"
)

// Expected extracts internal USB expectations from an already-resolved
// ODDC hardware view.
//
// ODDC owns hardware facts. This adapter contains no device-model,
// vendor, VID/PID, or machine-specific knowledge.
func Expected(resolved map[string]any) ([]usbtrust.ExpectedDevice, error) {
	if resolved == nil {
		return nil, nil
	}

	var result []usbtrust.ExpectedDevice

	if err := walk("", resolved, &result); err != nil {
		return nil, err
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Role < result[j].Role
	})

	return result, nil
}

func walk(
	path string,
	value any,
	result *[]usbtrust.ExpectedDevice,
) error {
	object, ok := value.(map[string]any)
	if !ok {
		return nil
	}

	bus, hasBus, err := optionalString(object, "bus")
	if err != nil {
		return fieldError(path, "bus", err)
	}

	attachment, hasAttachment, err := optionalString(
		object,
		"attachment",
	)
	if err != nil {
		return fieldError(path, "attachment", err)
	}

	if hasBus &&
		hasAttachment &&
		strings.EqualFold(bus, "usb") &&
		strings.EqualFold(attachment, "internal") {

		deviceID, hasDeviceID, err := optionalString(
			object,
			"deviceId",
		)
		if err != nil {
			return fieldError(path, "deviceId", err)
		}

		deviceID = strings.ToLower(strings.TrimSpace(deviceID))

		if !hasDeviceID || deviceID == "" {
			return fmt.Errorf(
				"ODDC internal USB component %q has no deviceId",
				path,
			)
		}

		required := true

		if raw, exists := object["required"]; exists {
			value, ok := raw.(bool)
			if !ok {
				return fmt.Errorf(
					"ODDC hardware field %q required must be boolean",
					path,
				)
			}

			required = value
		}

		*result = append(
			*result,
			usbtrust.ExpectedDevice{
				Role:     path,
				VIDPID:   deviceID,
				Required: required,
			},
		)
	}

	keys := make([]string, 0, len(object))

	for key := range object {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	for _, key := range keys {
		child, ok := object[key].(map[string]any)
		if !ok {
			continue
		}

		next := key
		if path != "" {
			next = path + "." + key
		}

		if err := walk(next, child, result); err != nil {
			return err
		}
	}

	return nil
}

func optionalString(
	object map[string]any,
	key string,
) (string, bool, error) {
	value, exists := object[key]
	if !exists {
		return "", false, nil
	}

	text, ok := value.(string)
	if !ok {
		return "", true, fmt.Errorf("must be a string")
	}

	return text, true, nil
}

func fieldError(
	path string,
	field string,
	err error,
) error {
	return fmt.Errorf(
		"ODDC hardware field %q %s: %w",
		path,
		field,
		err,
	)
}
