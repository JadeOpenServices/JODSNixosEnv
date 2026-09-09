package config

import (
	"path/filepath"
	"testing"
)

func TestDeviceIdentitySurvivesConfigRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "user.config.json")

	want := User{
		System:           "x86_64-linux",
		Profile:          "laptop",
		Hostname:         "testhost",
		Username:         "tester",
		Theme:            "noctalia",
		Shell:            "zsh",
		Editors:          []string{"vscodium"},
		Browsers:         []string{"librewolf"},
		PreferredEditor:  "vscodium",
		PreferredBrowser: "librewolf",

		DeviceProfile:        "laptop/common",
		DeviceSysVendor:      "HP",
		DeviceProductName:    "HP ZBook x2 G4",
		DeviceProductVersion: "A",
		DeviceBoardVendor:    "HP",
		DeviceBoardName:      "824C",
		DeviceBoardVersion:   "KBC Version 43.72",
	}

	if err := WriteAtomic(path, want); err != nil {
		t.Fatal(err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if got.DeviceProfile != want.DeviceProfile ||
		got.DeviceSysVendor != want.DeviceSysVendor ||
		got.DeviceProductName != want.DeviceProductName ||
		got.DeviceProductVersion != want.DeviceProductVersion ||
		got.DeviceBoardVendor != want.DeviceBoardVendor ||
		got.DeviceBoardName != want.DeviceBoardName ||
		got.DeviceBoardVersion != want.DeviceBoardVersion {
		t.Fatalf("device identity changed across roundtrip:\nwant=%+v\ngot=%+v", want, got)
	}
}
