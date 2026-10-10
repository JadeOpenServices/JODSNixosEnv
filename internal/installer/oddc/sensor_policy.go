package oddc

import (
	portable "github.com/JadeOpenServices/oddc/pkg/oddc"
)

// OrientationSensor reports whether the ODDC model has a sensor hub that
// provides screen orientation. ok is false when ODDC has no answer: no
// model, or the model leaves the field out.
func OrientationSensor(resolved Resolved) (present bool, ok bool) {
	if resolved.ModelID == "" ||
		resolved.Canonical.Resolved == nil {
		return false, false
	}

	value, found := portable.Lookup(
		resolved.Canonical.Resolved,
		"hardware.sensors.hub.provides.orientation",
	)
	if !found {
		return false, false
	}

	present, ok = value.(bool)
	return present, ok
}
