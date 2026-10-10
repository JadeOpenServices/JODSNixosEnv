package oddc_test

import (
	"testing"

	portable "github.com/JadeOpenServices/oddc/pkg/oddc"

	"github.com/JadeOpenServices/gjallarOS/internal/installer/oddc"
)

func TestOrientationSensor(t *testing.T) {
	withHub := func(provides map[string]any) oddc.Resolved {
		return oddc.Resolved{
			ModelID: "model",
			Canonical: portable.Resolved{Resolved: map[string]any{
				"hardware": map[string]any{
					"sensors": map[string]any{
						"hub": map[string]any{"provides": provides},
					},
				},
			}},
		}
	}

	for _, c := range []struct {
		name            string
		resolved        oddc.Resolved
		present, answer bool
	}{
		{"no model", oddc.Resolved{}, false, false},
		{"field missing", withHub(map[string]any{"accelerometer": true}), false, false},
		{"not a bool", withHub(map[string]any{"orientation": "yes"}), false, false},
		{"true", withHub(map[string]any{"orientation": true}), true, true},
		{"false", withHub(map[string]any{"orientation": false}), false, true},
	} {
		present, ok := oddc.OrientationSensor(c.resolved)
		if present != c.present || ok != c.answer {
			t.Errorf("%s: got (%t, %t), want (%t, %t)",
				c.name, present, ok, c.present, c.answer)
		}
	}
}
