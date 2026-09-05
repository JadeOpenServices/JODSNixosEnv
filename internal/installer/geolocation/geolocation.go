// Package geolocation detects an approximate city and country from the
// installer's public network address. It never treats detection as consent.
package geolocation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const endpoint = "https://ipwho.is/"

type Location struct {
	City    string
	Country string
}

type response struct {
	Success bool   `json:"success"`
	City    string `json:"city"`
	Country string `json:"country"`
	Message string `json:"message"`
}

func Detect(ctx context.Context) (Location, error) {
	client := &http.Client{Timeout: 8 * time.Second}
	return detect(ctx, client, endpoint)
}

func detect(ctx context.Context, client *http.Client, url string) (Location, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Location{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "GjallarOS-installer/1")
	res, err := client.Do(req)
	if err != nil {
		return Location{}, fmt.Errorf("network location request: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return Location{}, fmt.Errorf("network location request returned HTTP %d", res.StatusCode)
	}
	var payload response
	decoder := json.NewDecoder(res.Body)
	if err := decoder.Decode(&payload); err != nil {
		return Location{}, fmt.Errorf("decode network location: %w", err)
	}
	payload.City = strings.TrimSpace(payload.City)
	payload.Country = strings.TrimSpace(payload.Country)
	if !payload.Success || payload.City == "" || payload.Country == "" {
		return Location{}, fmt.Errorf("network location unavailable: %s", strings.TrimSpace(payload.Message))
	}
	return Location{City: payload.City, Country: payload.Country}, nil
}
