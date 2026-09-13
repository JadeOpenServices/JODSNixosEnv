package discovery

import (
	"github.com/bakanura/gjallarOS/internal/hardware/inputclass"
	"github.com/bakanura/gjallarOS/internal/hardware/orientation"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Hardware struct {
	FormFactor   string
	LaptopVendor string

	SysVendor      string
	ProductName    string
	ProductVersion string
	BoardVendor    string
	BoardName      string
	BoardVersion   string

	Touchscreen       bool
	PenTablet         bool
	OrientationSensor bool
}
type Options struct{ Profiles, Shells, Editors, Browsers, Themes []string }

func DetectHardware(sysRoot string) Hardware {
	h := Hardware{
		SysVendor:      readDMI(sysRoot, "sys_vendor"),
		ProductName:    readDMI(sysRoot, "product_name"),
		ProductVersion: readDMI(sysRoot, "product_version"),
		BoardVendor:    readDMI(sysRoot, "board_vendor"),
		BoardName:      readDMI(sysRoot, "board_name"),
		BoardVersion:   readDMI(sysRoot, "board_version"),
	}

	batteries, _ := filepath.Glob(filepath.Join(sysRoot, "class", "power_supply", "BAT*"))
	if len(batteries) > 0 {
		h.FormFactor = "laptop"
		h.LaptopVendor = "generic"
	} else if data, err := os.ReadFile(filepath.Join(sysRoot, "class", "dmi", "id", "chassis_type")); err == nil {
		switch strings.TrimSpace(string(data)) {
		case "8", "9", "10", "11", "14", "30", "31", "32":
			h.FormFactor = "laptop"
			h.LaptopVendor = "generic"
		case "3", "4", "5", "6", "7", "13", "15", "16", "17", "18", "19", "20", "21", "22", "23", "24", "25", "26", "27", "28", "29", "33", "34", "35", "36":
			h.FormFactor = "desktop"
		}
	}

	if h.ProductName != "" {
		product := strings.ToLower(h.ProductName)
		if strings.Contains(product, "thinkpad") {
			h.FormFactor = "laptop"
			h.LaptopVendor = "thinkpad"
		} else if strings.Contains(product, "framework") {
			h.FormFactor = "laptop"
			h.LaptopVendor = "framework"
		}
	}

	h.Touchscreen = detectTouchscreen(sysRoot)
	h.PenTablet = detectPenTablet(sysRoot)
	h.OrientationSensor = detectOrientationSensor(sysRoot)
	return h
}

func readDMI(sysRoot, name string) string {
	data, err := os.ReadFile(filepath.Join(sysRoot, "class", "dmi", "id", name))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func detectPenTablet(sysRoot string) bool {
	devices, _ := filepath.Glob(filepath.Join(sysRoot, "class", "input", "event*", "device"))
	for _, device := range devices {
		if inputclass.IsPen(inputclass.Device{
			Name: readInput(device, "name"),
			Key:  readInput(device, "capabilities", "key"),
			Abs:  readInput(device, "capabilities", "abs"),
		}) {
			return true
		}
	}
	return false
}

func detectTouchscreen(sysRoot string) bool {
	devices, _ := filepath.Glob(filepath.Join(sysRoot, "class", "input", "event*", "device"))
	for _, device := range devices {
		if inputclass.IsTouchscreen(inputclass.Device{
			Name:       readInput(device, "name"),
			Properties: readInput(device, "properties"),
			Abs:        readInput(device, "capabilities", "abs"),
		}) {
			return true
		}
	}
	return false
}

func detectOrientationSensor(sysRoot string) bool {
	devices, _ := filepath.Glob(
		filepath.Join(sysRoot, "bus", "iio", "devices", "iio:device*"),
	)

	for _, device := range devices {
		nameData, _ := os.ReadFile(filepath.Join(device, "name"))
		name := strings.TrimSpace(string(nameData))

		entries, _ := filepath.Glob(filepath.Join(device, "in_*"))
		channels := make([]string, 0, len(entries))
		for _, entry := range entries {
			channels = append(channels, filepath.Base(entry))
		}

		if orientation.IsSensor(name, channels) {
			return true
		}
	}

	return false
}

func readInput(device string, parts ...string) string {
	path := filepath.Join(append([]string{device}, parts...)...)
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// Linux exposes input capability bitsets as hexadecimal, most-significant
// word first. Reading from the right keeps this independent of word size.
func Discover(repo string, preset bool, hardware Hardware) (Options, error) {
	profiles, err := directories(filepath.Join(repo, "profiles"))
	if err != nil {
		return Options{}, err
	}
	if !preset {
		filtered := profiles[:0]
		for _, v := range profiles {
			if hardware.FormFactor == "desktop" {
				if v == "desktop" {
					filtered = append(filtered, v)
				}
			} else if v != "desktop" && v != "work" && v != "work-user" {
				filtered = append(filtered, v)
			}
		}
		profiles = filtered
	}
	shells, err := nixFiles(filepath.Join(repo, "user", "shells"))
	if err != nil {
		return Options{}, err
	}
	editors, err := directories(filepath.Join(repo, "user", "editors"))
	if err != nil {
		return Options{}, err
	}
	browsers, err := nixFiles(filepath.Join(repo, "user", "browsers"))
	if err != nil {
		return Options{}, err
	}
	themes, err := nixFiles(filepath.Join(repo, "themes"))
	if err != nil {
		return Options{}, err
	}
	return Options{profiles, shells, editors, browsers, themes}, nil
}

func directories(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	out := []string{}
	for _, entry := range entries {
		if entry.IsDir() {
			out = append(out, entry.Name())
		}
	}
	sort.Strings(out)
	return out, nil
}
func nixFiles(path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	out := []string{}
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".nix" {
			out = append(out, strings.TrimSuffix(entry.Name(), ".nix"))
		}
	}
	sort.Strings(out)
	return out, nil
}
