package app

import "testing"

func TestMayOfferSecureBootFallback(t *testing.T) {
	tests := []struct {
		name       string
		enabled    bool
		managed    bool
		unattended bool
		want       bool
	}{
		{
			name:    "interactive unmanaged unsupported request may downgrade",
			enabled: true,
			want:    true,
		},
		{
			name:    "managed must fail closed",
			enabled: true,
			managed: true,
			want:    false,
		},
		{
			name:       "unattended must fail closed",
			enabled:    true,
			unattended: true,
			want:       false,
		},
		{
			name:    "interactive config file remains interactive",
			enabled: true,
			want:    true,
		},
		{
			name: "disabled secure boot needs no fallback",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mayOfferSecureBootFallback(
				tt.enabled,
				tt.managed,
				tt.unattended,
			)
			if got != tt.want {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTPM2AllowedOnlyWithSecureBoot(t *testing.T) {
	if tpm2AllowedForSecureBoot(false) {
		t.Fatal("TPM2 measured-boot unlock allowed while Secure Boot disabled")
	}

	if !tpm2AllowedForSecureBoot(true) {
		t.Fatal("TPM2 measured-boot unlock blocked while Secure Boot enabled")
	}
}
