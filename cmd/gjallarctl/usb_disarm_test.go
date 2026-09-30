package main

import (
	"strings"
	"testing"
)

func TestRecoveryKeyKeepsInnerWhitespace(t *testing.T) {
	for input, want := range map[string]string{
		"correct horse \n": "correct horse ",
		"windows line\r\n": "windows line",
		"no newline":       "no newline",
		"first\nsecond\n":  "first",
		" leading space\n": " leading space",
	} {
		got, err := usbParseRecoveryKey(strings.NewReader(input))
		if err != nil || got != want {
			t.Fatalf("%q: got %q, %v; want %q", input, got, err, want)
		}
	}
}

func TestRecoveryKeyRejectsEmptyInput(t *testing.T) {
	for _, input := range []string{"", "\n", "\r\n"} {
		if _, err := usbParseRecoveryKey(strings.NewReader(input)); err == nil {
			t.Fatalf("%q: accepted an empty passphrase", input)
		}
	}
}
