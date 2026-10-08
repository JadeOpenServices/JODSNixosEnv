// Package profile selects a local-AI profile from hardware and user override.
package profile

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/JadeOpenServices/gjallarOS/internal/hardware/graphics"
)

type Hardware struct {
	RAMGB    int
	CPUCores int
	Arch     string
	GPUType  string
	VRAMMB   int
}

type Override struct {
	Enabled bool
	Model   string
}

type Result struct {
	Profile             string `json:"profile"`
	AccelerationProfile string `json:"accelerationProfile"`
	Model               string `json:"model"`
	ContextTokens       int    `json:"contextTokens"`
	VRAMMB              int    `json:"vramMB"`
	RAMGB               int    `json:"ramGB"`
	GPUVendor           string `json:"gpuVendor"`
	GPUType             string `json:"gpuType"`
	CPUCores            int    `json:"cpuCores"`
	Architecture        string `json:"architecture"`
}

type modelProfile struct {
	Name, Model                                     string
	ContextTokens, MinRAMGB, MinCPUCores, MinVRAMMB int
	DedicatedGPU                                    bool
}

// modelProfiles is the single authoritative automatic-selection table. Order
// is strongest to weakest; the final entry is the safe fallback.
var modelProfiles = []modelProfile{
	{Name: "dedicated", Model: "qwen3-coder:30b", ContextTokens: 32768, MinRAMGB: 32, MinCPUCores: 8, MinVRAMMB: 12288, DedicatedGPU: true},
	{Name: "integrated", Model: "qwen2.5-coder:14b", ContextTokens: 16384, MinRAMGB: 16, MinCPUCores: 4},
	{Name: "low-memory", Model: "qwen2.5-coder:7b", ContextTokens: 8192},
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
	result.CPUCores, result.Architecture = hardware.CPUCores, hardware.Arch
	return result, nil
}

func accelerationProfileForHardware(h Hardware, selectedProfile string) string {
	if selectedProfile == "dedicated" {
		return "full"
	}

	if selectedProfile == "integrated" && h.RAMGB >= 32 {
		return "full"
	}

	return "auto"
}

func Select(hardware Hardware, override Override) Result {
	selected := modelProfiles[len(modelProfiles)-1]
	for _, candidate := range modelProfiles {
		if hardware.RAMGB < candidate.MinRAMGB || hardware.CPUCores < candidate.MinCPUCores || hardware.VRAMMB < candidate.MinVRAMMB {
			continue
		}
		if candidate.DedicatedGPU && hardware.GPUType != "dedicated" {
			continue
		}
		selected = candidate
		break
	}
	result := Result{Profile: selected.Name, Model: selected.Model, ContextTokens: selected.ContextTokens, VRAMMB: hardware.VRAMMB}
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
	result.AccelerationProfile =
		accelerationProfileForHardware(hardware, result.Profile)
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
	if config.Enabled && model == "" {
		return Override{}, fmt.Errorf("overrideModelWith is required when overrideAiSelection is true")
	}
	if model != "" && (strings.ContainsAny(model, " \t") || strings.HasPrefix(model, "-") || strings.Contains(model, "/../")) {
		return Override{}, fmt.Errorf("overrideModelWith is not a valid Ollama model identifier")
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
		return Hardware{RAMGB: ramGB, CPUCores: runtime.NumCPU(), Arch: runtime.GOARCH, GPUType: "unknown", VRAMMB: vramMB()}, "unknown", nil
	}
	gpuType := live.Type
	if gpuType == "hybrid" {
		// Preserve the installer policy: a multi-GPU machine may use the
		// dedicated profile when its discrete VRAM is sufficient.
		gpuType = "dedicated"
	}
	return Hardware{RAMGB: ramGB, CPUCores: runtime.NumCPU(), Arch: runtime.GOARCH, GPUType: gpuType, VRAMMB: vramMB()}, live.Vendor, nil
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
