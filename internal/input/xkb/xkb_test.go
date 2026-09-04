package xkb

import "testing"

func TestNormalize(t *testing.T) {
	tests := map[string]string{
		"":          "de",
		"DE":        "de",
		"de-latin1": "de",
		"us":        "us",
	}
	for input, want := range tests {
		got, err := Normalize(input)
		if err != nil || got.Name != want || got.Variant != "" {
			t.Fatalf("Normalize(%q) = %#v, %v", input, got, err)
		}
	}
}

func TestNormalizeRejectsUnsafeInput(t *testing.T) {
	if _, err := Normalize("de; rm"); err == nil {
		t.Fatal("unsafe layout must fail")
	}
}
