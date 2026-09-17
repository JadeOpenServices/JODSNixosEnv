package usbtrust

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

type testSigner struct {
	public  ed25519.PublicKey
	private ed25519.PrivateKey
}

func newTestSigner(t *testing.T) *testSigner {
	t.Helper()

	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	return &testSigner{
		public:  public,
		private: private,
	}
}

func (s *testSigner) Sign(
	_ context.Context,
	payload []byte,
) (Signature, error) {
	digest := sha256.Sum256(payload)
	signature := ed25519.Sign(s.private, payload)

	return Signature{
		Schema:       1,
		Algorithm:    "test-ed25519",
		KeyReference: "test",
		DigestSHA256: hex.EncodeToString(digest[:]),
		Value:        base64.StdEncoding.EncodeToString(signature),
	}, nil
}

func (s *testSigner) Verify(
	_ context.Context,
	payload []byte,
	signature Signature,
) error {
	raw, err := base64.StdEncoding.DecodeString(signature.Value)
	if err != nil {
		return err
	}

	if !ed25519.Verify(s.public, payload, raw) {
		return os.ErrPermission
	}

	return nil
}

func sampleDocument() Document {
	return Document{
		Schema:    SchemaVersion,
		MachineID: "machine-test",
		ODDCModel: "model/framework/laptop-13-amd-ryzen-7040",
		Revision:  1,
		Devices: []Device{
			{
				ID:             "internal:fingerprint",
				Role:           "fingerprint",
				Class:          ClassInternal,
				ExpectedByODDC: true,
				Strength:       StrengthSerialDescriptorTopology,
				Identity: Identity{
					VIDPID:      "27c6:609c",
					Serial:      "example",
					Hash:        "hash-example",
					ParentHash:  "parent-example",
					Interfaces:  []string{"ff:00:00"},
					ConnectType: "hardwired",
				},
				FirstAccepted: "2026-09-17T00:00:00Z",
				LastAccepted:  "2026-09-17T00:00:00Z",
			},
		},
	}
}

func TestSignedStoreRoundTrip(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state")
	signer := newTestSigner(t)

	if err := WriteSigned(
		context.Background(),
		dir,
		sampleDocument(),
		signer,
	); err != nil {
		t.Fatal(err)
	}

	got, err := ReadVerified(
		context.Background(),
		dir,
		signer,
	)
	if err != nil {
		t.Fatal(err)
	}

	if got.ODDCModel != "model/framework/laptop-13-amd-ryzen-7040" {
		t.Fatalf("unexpected model %q", got.ODDCModel)
	}
}

func TestTamperedTrustDocumentFailsVerification(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state")
	signer := newTestSigner(t)

	if err := WriteSigned(
		context.Background(),
		dir,
		sampleDocument(),
		signer,
	); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(dir, TrustFile)

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	for i := range data {
		if data[i] == '6' {
			data[i] = '7'
			break
		}
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := ReadVerified(
		context.Background(),
		dir,
		signer,
	); err == nil {
		t.Fatal("tampered USB trust document verified successfully")
	}
}

func TestCanonicalOrderingIsStable(t *testing.T) {
	doc := sampleDocument()

	doc.Devices[0].Identity.Interfaces = []string{
		"ff:02:00",
		"ff:00:00",
	}

	second := doc.Devices[0]
	second.ID = "external:test"
	second.Class = ClassExternal
	second.ExpectedByODDC = false
	second.Portable = true

	doc.Devices = append([]Device{second}, doc.Devices...)

	first, err := Canonical(doc)
	if err != nil {
		t.Fatal(err)
	}

	secondBytes, err := Canonical(doc)
	if err != nil {
		t.Fatal(err)
	}

	if string(first) != string(secondBytes) {
		t.Fatal("canonical USB trust serialization is unstable")
	}
}
