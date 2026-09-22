package usbtrust

import "testing"

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
