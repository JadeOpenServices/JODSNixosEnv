// Package profile selects a local-AI profile from hardware and user override.
package profile

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/bakanura/gjallarOS/internal/hardware/graphics"
)

type Hardware struct {
	RAMGB   int
	GPUType string
	VRAMMB  int
}

type Override struct {
	Enabled bool
	Model   string
}

type Result struct {
	Profile       string `json:"profile"`
	Model         string `json:"model"`
	ContextTokens int    `json:"contextTokens"`
	VRAMMB        int    `json:"vramMB"`
	RAMGB         int    `json:"ramGB"`
	GPUVendor     string `json:"gpuVendor"`
	GPUType       string `json:"gpuType"`
}

func Detect(ctx context.Context, configPath string) (Result, error) {
	hardware, vendor, err := detectHardware(ctx)
	if err != nil {
		return Result{}, err
	}
	override, err := LoadOverride(configPath)
	if err != nil {
		return Result{}, err
	}
	result := Select(hardware, override)
	result.RAMGB, result.GPUVendor, result.GPUType = hardware.RAMGB, vendor, hardware.GPUType
	return result, nil
}

func Select(hardware Hardware, override Override) Result {
	result := Result{Profile: "low-memory", Model: "qwen3-coder:7b", ContextTokens: 8192, VRAMMB: hardware.VRAMMB}
	if hardware.RAMGB >= 32 && hardware.GPUType == "dedicated" && hardware.VRAMMB >= 12288 {
		result.Profile, result.Model, result.ContextTokens = "dedicated", "qwen3-coder:30b", 32768
	} else if hardware.RAMGB >= 16 {
		result.Profile, result.Model, result.ContextTokens = "integrated", "qwen3-coder:14b", 16384
	}
	if override.Enabled && override.Model != "" {
		result.Profile, result.Model = "user-override", override.Model
		switch {
		case hardware.RAMGB >= 32:
			result.ContextTokens = 32768
		case hardware.RAMGB >= 16:
			result.ContextTokens = 16384
		default:
			result.ContextTokens = 8192
		}
	}
	return result
}

func LoadOverride(path string) (Override, error) {
	if path == "" {
		return Override{}, nil
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Override{}, nil
		}
		return Override{}, fmt.Errorf("read user configuration: %w", err)
	}
	var config struct {
		Enabled bool   `json:"overrideAiSelection"`
		Model   string `json:"overrideModelWith"`
	}
	if err := json.Unmarshal(contents, &config); err != nil {
		return Override{}, fmt.Errorf("parse user configuration: %w", err)
	}
	model := strings.TrimSpace(config.Model)
	if strings.ContainsAny(model, "\x00\r\n") {
		return Override{}, fmt.Errorf("overrideModelWith must not contain control characters")
	}
	return Override{Enabled: config.Enabled, Model: model}, nil
}

func detectHardware(ctx context.Context) (Hardware, string, error) {
	ramGB, err := memoryGB("/proc/meminfo")
	if err != nil {
		return Hardware{}, "", err
	}
	live, err := graphics.Detect(ctx)
	if err != nil {
		// Local AI remains usable without lspci; retain the low-memory fallback.
		return Hardware{RAMGB: ramGB, GPUType: "unknown", VRAMMB: vramMB()}, "unknown", nil
	}
	gpuType := live.Type
	if gpuType == "hybrid" {
		// Preserve the installer policy: a multi-GPU machine may use the
		// dedicated profile when its discrete VRAM is sufficient.
		gpuType = "dedicated"
	}
	return Hardware{RAMGB: ramGB, GPUType: gpuType, VRAMMB: vramMB()}, live.Vendor, nil
}

func memoryGB(path string) (int, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("read memory information: %w", err)
	}
	for _, line := range strings.Split(string(contents), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[0] == "MemTotal:" {
			kilobytes, err := strconv.Atoi(fields[1])
			if err != nil || kilobytes < 0 {
				return 0, fmt.Errorf("parse MemTotal")
			}
			return kilobytes / 1024 / 1024, nil
		}
	}
	return 0, fmt.Errorf("MemTotal is missing")
}

func vramMB() int {
	paths, err := filepath.Glob("/sys/class/drm/card*/device/mem_info_vram_total")
	if err != nil {
		return 0
	}
	maximum := 0
	for _, path := range paths {
		contents, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		bytes, err := strconv.ParseInt(strings.TrimSpace(string(contents)), 10, 64)
		if err == nil && bytes > 0 {
			maximum = max(maximum, int(bytes/(1024*1024)))
		}
	}
	return maximum
}
