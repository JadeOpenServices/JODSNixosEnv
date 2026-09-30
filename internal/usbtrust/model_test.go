package usbtrust

import (
	"strings"
	"testing"
)

func TestDocumentRejectsDuplicateInternalRole(t *testing.T) {
	first := internalTrusted()
	second := first
	second.ID = "internal:fingerprint-copy"
	doc := *trustedDocument(first, second)
	if err := doc.Validate(); err == nil {
		t.Fatal("accepted duplicate internal ODDC role")
	}
}

func TestDocumentRejectsExternalODDCRole(t *testing.T) {
	device := syntheticDock()
	device.Role = "hardware.fake.internal"
	device.ExpectedByODDC = true
	doc := *trustedDocument(device)
	if err := doc.Validate(); err == nil {
		t.Fatal("accepted external device claiming ODDC role")
	}
}

func TestDocumentRejectsUnknownIdentityStrength(t *testing.T) {
	device := syntheticDock()
	device.Strength = IdentityStrength("mystery")
	doc := *trustedDocument(device)
	if err := doc.Validate(); err == nil {
		t.Fatal("accepted unknown identity strength")
	}
}

func TestLegacyDocumentCanonicalFormIsUnchanged(t *testing.T) {
	// Existing TPM signatures cover documents without enforcement state.
	payload, err := Canonical(*trustedDocument(internalTrusted()))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(payload), "enforcement") {
		t.Fatalf("legacy canonical form changed: %s", payload)
	}
}

func TestDocumentRejectsUndatedEnforcement(t *testing.T) {
	doc := *trustedDocument(internalTrusted())
	doc.Enforcement = &Enforcement{Armed: true}
	if err := doc.Validate(); err == nil {
		t.Fatal("accepted enforcement state without times")
	}
}
