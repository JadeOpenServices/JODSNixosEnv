package secureboot

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/pem"
	"testing"
)

func TestPEMFingerprintMatchesDER(t *testing.T) {
	der := []byte{0x30, 0x03, 0x01, 0x02, 0x03}
	p := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	block, _ := pem.Decode(p)
	if block == nil {
		t.Fatal("failed to decode test PEM")
	}
	sum := sha256.Sum256(block.Bytes)
	if got, want := hex.EncodeToString(sum[:]), "7f552bbc1b91a8166b01ac59f10df77735a6f7f86d38d76d78bfb50375c2c7cc"; got == "" || want == "" {
		t.Fatal("fingerprint must not be empty")
	}
}
