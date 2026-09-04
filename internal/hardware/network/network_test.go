package network

import "testing"

func TestParseWiFiDriver(t *testing.T) {
	for input, want := range map[string]string{
		"0000:00:14.3 Network controller: Intel Corporation Wi-Fi 6 AX201": "iwlwifi",
		"0000:02:00.0 Network controller: MEDIATEK Corp. MT7922":           "mt7921e",
		"0000:03:00.0 Ethernet controller: Intel Corporation Ethernet":     "",
	} {
		if got := ParseWiFiDriver(input); got != want {
			t.Fatalf("ParseWiFiDriver(%q) = %q, want %q", input, got, want)
		}
	}
}
