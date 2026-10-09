package config

import (
	"reflect"
	"testing"
)

// The preset file is a template for automated installs. A value only the
// file sets would give automated installs something interactive installs
// never get (secureBootPrompt, real HP, 2026-10-09).
func TestDefaultsMatchPresetFile(t *testing.T) {
	preset, err := Load("../../../scripts/installation/user_PresetJSON/default.user.config.json")
	if err != nil {
		t.Fatal(err)
	}
	defaults, err := Defaults()
	if err != nil {
		t.Fatal(err)
	}
	if err := Validate(defaults); err != nil {
		t.Fatalf("Defaults() is not a valid configuration: %v", err)
	}
	want := reflect.ValueOf(defaults)
	got := reflect.ValueOf(preset)
	for i := 0; i < want.NumField(); i++ {
		field := want.Type().Field(i)
		if !reflect.DeepEqual(want.Field(i).Interface(), got.Field(i).Interface()) {
			t.Errorf("%s: Defaults() has %#v, default.user.config.json has %#v",
				field.Tag.Get("json"), want.Field(i).Interface(), got.Field(i).Interface())
		}
	}
}
