package app

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/bakanura/gjallarOS/internal/installer/config"
	"github.com/bakanura/gjallarOS/internal/installer/prompt"
)

func TestConfigureWeatherLocationUsesConfiguredValues(t *testing.T) {
	u := &config.User{
		WeatherCity:    "  Frankfurt am Main  ",
		WeatherCountry: "  Germany  ",
	}

	var out bytes.Buffer

	if err := configureWeatherLocation(
		context.Background(),
		prompt.UI{},
		u,
		&out,
	); err != nil {
		t.Fatalf("configureWeatherLocation: %v", err)
	}

	if u.WeatherCity != "Frankfurt am Main" {
		t.Fatalf("WeatherCity = %q", u.WeatherCity)
	}
	if u.WeatherCountry != "Germany" {
		t.Fatalf("WeatherCountry = %q", u.WeatherCountry)
	}

	if !strings.Contains(
		out.String(),
		"Weather location configured: Frankfurt am Main, Germany",
	) {
		t.Fatalf("unexpected output: %q", out.String())
	}
}
