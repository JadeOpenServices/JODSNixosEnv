package secureboot

import "testing"

func TestGeneratePassphrase(t *testing.T) {
	a, err := generatePassphrase()
	if err != nil || len(a) != 64 {
		t.Fatalf("generatePassphrase() = %q, %v", a, err)
	}
	b, err := generatePassphrase()
	if err != nil || a == b {
		t.Fatal("recovery passphrases must be unique")
	}
}
