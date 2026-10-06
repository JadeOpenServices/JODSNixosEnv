package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOrientationTransform(t *testing.T) {
	cases := map[string]int{
		"normal":    0,
		"right-up":  1,
		"bottom-up": 2,
		"left-up":   3,
	}

	for input, want := range cases {
		got, ok := orientationTransform(input)
		if !ok || got != want {
			t.Fatalf("%q => %d,%v want %d,true", input, got, ok, want)
		}
	}

	if _, ok := orientationTransform("unknown"); ok {
		t.Fatal("unknown orientation unexpectedly accepted")
	}
}

func TestInternalDRMConnectorFindsConnectedEDP(t *testing.T) {
	root := t.TempDir()

	disconnected := filepath.Join(root, "class", "drm", "card0-eDP-1")
	connected := filepath.Join(root, "class", "drm", "card1-eDP-2")

	if err := os.MkdirAll(disconnected, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(connected, 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(disconnected, "status"),
		[]byte("disconnected\n"),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(connected, "status"),
		[]byte("connected\n"),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	got, err := internalDRMConnector(root)
	if err != nil {
		t.Fatal(err)
	}
	if got != "eDP-2" {
		t.Fatalf("got %q want eDP-2", got)
	}
}
