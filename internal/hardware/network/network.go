// Package network discovers the Wi-Fi driver family without changing hardware.
package network

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func DetectWiFiDriver(ctx context.Context) (string, error) {
	lspci, err := exec.LookPath("lspci")
	if err != nil {
		return "", nil
	}
	output, err := exec.CommandContext(ctx, lspci, "-Dnn").Output()
	if err != nil {
		return "", fmt.Errorf("run lspci: %w", err)
	}
	return ParseWiFiDriver(string(output)), nil
}

func ParseWiFiDriver(output string) string {
	for _, line := range strings.Split(output, "\n") {
		lower := strings.ToLower(line)
		if !strings.Contains(lower, "network controller") && !strings.Contains(lower, "wireless controller") {
			continue
		}
		switch {
		case strings.Contains(lower, "intel"):
			return "iwlwifi"
		case strings.Contains(lower, "qualcomm"), strings.Contains(lower, "atheros"):
			return "ath10k_pci"
		case strings.Contains(lower, "mediatek"):
			return "mt7921e"
		case strings.Contains(lower, "realtek"):
			return "rtw89pci"
		case strings.Contains(lower, "broadcom"):
			return "brcmfmac"
		}
	}
	return ""
}
