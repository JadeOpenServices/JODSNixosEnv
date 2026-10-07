package secureboot

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
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

func signatureList(owner byte, data []byte) []byte {
	x509Type := []byte{0xa1, 0x59, 0xc0, 0xa5, 0xe4, 0x94, 0xa7, 0x4a, 0x87, 0xb5, 0xab, 0x15, 0x5c, 0x2b, 0xf0, 0x72}
	size := 16 + len(data)
	list := append([]byte{}, x509Type...)
	list = binary.LittleEndian.AppendUint32(list, uint32(28+size))
	list = binary.LittleEndian.AppendUint32(list, 0)
	list = binary.LittleEndian.AppendUint32(list, uint32(size))
	list = append(list, bytes.Repeat([]byte{owner}, 16)...)
	return append(list, data...)
}

func TestSamePlatformKeyIgnoresSignatureOwner(t *testing.T) {
	cert := []byte("0\x82\x03/ der certificate bytes")
	if !samePlatformKey(signatureList(1, cert), signatureList(2, cert)) {
		t.Fatal("same certificate under another owner GUID was treated as a different PK")
	}
	if samePlatformKey(signatureList(1, cert), signatureList(1, append(cert, 'x'))) {
		t.Fatal("different certificates were treated as the same PK")
	}
	if samePlatformKey([]byte("short"), signatureList(1, cert)) {
		t.Fatal("unparsable list matched")
	}
}
