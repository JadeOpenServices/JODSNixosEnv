package discovery

import "testing"

func TestODDCIdentityPreservesStructuredHardwareIdentity(t *testing.T) {
	hardware := Hardware{
		FormFactor:     "laptop",
		SysVendor:      "HP",
		ProductName:    "HP ZBook x2 G4",
		ProductVersion: "A",
		BoardVendor:    "HP",
		BoardName:      "824C",
		BoardVersion:   "KBC Version 43.72",
	}

	got := ODDCIdentity(hardware)

	if got.FormFactor != hardware.FormFactor {
		t.Fatalf("FormFactor=%q", got.FormFactor)
	}
	if got.SysVendor != hardware.SysVendor {
		t.Fatalf("SysVendor=%q", got.SysVendor)
	}
	if got.ProductName != hardware.ProductName {
		t.Fatalf("ProductName=%q", got.ProductName)
	}
	if got.ProductVersion != hardware.ProductVersion {
		t.Fatalf("ProductVersion=%q", got.ProductVersion)
	}
	if got.BoardVendor != hardware.BoardVendor {
		t.Fatalf("BoardVendor=%q", got.BoardVendor)
	}
	if got.BoardName != hardware.BoardName {
		t.Fatalf("BoardName=%q", got.BoardName)
	}
	if got.BoardVersion != hardware.BoardVersion {
		t.Fatalf("BoardVersion=%q", got.BoardVersion)
	}
}
