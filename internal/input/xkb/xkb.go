// Package xkb normalizes installer keyboard-layout input.
package xkb

import (
	"fmt"
	"regexp"
	"strings"
)

type Layout struct {
	Name    string
	Variant string
}

var layoutName = regexp.MustCompile(`^[a-z0-9,_+-]+$`)

func Normalize(requested string) (Layout, error) {
	name := strings.ToLower(strings.TrimSpace(requested))
	if name == "" {
		name = "de"
	}
	if !layoutName.MatchString(name) {
		return Layout{}, fmt.Errorf("keyboard layout may contain only letters, numbers, commas, plus, underscore, and hyphen")
	}
	// xkeyboard-config no longer provides de(latin1). The regular German map
	// is the supported replacement and works in both SDDM and Hyprland.
	if name == "de" || name == "de-latin1" {
		name = "de"
	}
	return Layout{Name: name}, nil
}
