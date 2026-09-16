package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFanStrategyAllowed(t *testing.T) {
	cfg := fanRuntimeConfig{
		Strategies: []string{"balanced", "quiet"},
	}

	if !fanStrategyAllowed(cfg, "quiet") {
		t.Fatal("quiet strategy rejected")
	}
	if fanStrategyAllowed(cfg, "turbo") {
		t.Fatal("unknown strategy accepted")
	}
}

func TestLoadFanRuntimeConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "fan-control.json")

	data := []byte(`{
	  "schema": 1,
	  "enabled": true,
	  "backend": "fw-fanctrl",
	  "profile": "13",
	  "defaultStrategy": "balanced",
	  "strategyOnDischarging": "quiet",
	  "strategies": ["balanced", "quiet"]
	}`)

	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := loadFanRuntimeConfig(path)
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Backend != "fw-fanctrl" {
		t.Fatalf("backend=%q", cfg.Backend)
	}
	if cfg.Profile != "13" {
		t.Fatalf("profile=%q", cfg.Profile)
	}
	if !fanStrategyAllowed(cfg, "balanced") {
		t.Fatal("balanced strategy missing")
	}
}

func TestChooseFanStrategySystemQuietHysteresis(t *testing.T) {
	cfg := fanRuntimeConfig{
		DefaultStrategy: "balanced",
		Strategies: []string{
			"balanced",
			"cooling",
			"max",
			"performance",
			"quiet",
		},
		SystemPolicy: fanSystemPolicy{
			QuietStrategy:           "quiet",
			QuietEnterC:             60,
			QuietExitC:              68,
			PerformanceStrategy:     "performance",
			ThermalOverrideStrategy: "max",
			ThermalEnterC:           82,
			ThermalExitC:            74,
		},
	}

	tests := []struct {
		name    string
		current string
		temp    int
		profile string
		want    string
		reason  string
	}{
		{"enter quiet below threshold", "performance", 59, "performance", "quiet", "temperature"},
		{"enter quiet at threshold", "balanced", 60, "balanced", "quiet", "temperature"},
		{"hold quiet through hysteresis", "quiet", 67, "performance", "quiet", "temperature"},
		{"leave quiet at exit threshold", "quiet", 68, "performance", "performance", "power-profile"},
		{"normal balanced policy", "balanced", 70, "balanced", "balanced", "system"},
		{"thermal override wins", "quiet", 82, "balanced", "max", "thermal"},
		{"thermal hysteresis holds", "max", 75, "balanced", "max", "thermal"},
		{"thermal hysteresis exits", "max", 74, "balanced", "balanced", "system"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, reason := chooseFanStrategy(
				cfg,
				tt.current,
				tt.temp,
				tt.profile,
			)

			if got != tt.want || reason != tt.reason {
				t.Fatalf(
					"got (%q,%q), want (%q,%q)",
					got, reason, tt.want, tt.reason,
				)
			}
		})
	}
}
