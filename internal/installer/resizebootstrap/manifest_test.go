package resizebootstrap

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"strings"
	"testing"

	"github.com/bakanura/gjallarOS/internal/installer/recoveryresize"
)

func manifest() recoveryresize.Manifest {
	return recoveryresize.Manifest{
		Version: recoveryresize.ManifestVersion,

		Nonce: "nonce-123",
		Stage: recoveryresize.StagePreparingResize,

		DiskGUID:      "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee",
		DiskSizeBytes: 1000 * 1024 * 1024 * 1024,

		RootPARTUUID:   "11111111-2222-3333-4444-555555555555",
		RootStartBytes: 1024 * 1024 * 1024,

		ExpectedCurrentRootEndBytes: 901 * 1024 * 1024 * 1024,

		ExpectedNewRootEndBytes: 896 * 1024 * 1024 * 1024,

		LUKSUUID:      "22222222-3333-4444-5555-666666666666",
		BtrfsUUID:     "33333333-4444-5555-6666-777777777777",
		RecoveryBytes: 12 * 1024 * 1024 * 1024,
	}
}

func keypair(t *testing.T) (
	ed25519.PublicKey,
	ed25519.PrivateKey,
) {
	t.Helper()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	return pub, priv
}

func TestSignVerifyManifest(t *testing.T) {
	pub, priv := keypair(t)

	envelope, err := Sign(manifest(), priv)
	if err != nil {
		t.Fatal(err)
	}

	got, err := Verify(envelope, pub)
	if err != nil {
		t.Fatal(err)
	}

	if got.Nonce != "nonce-123" {
		t.Fatalf("nonce=%q", got.Nonce)
	}
	if got.RootStartBytes != manifest().RootStartBytes {
		t.Fatal("root start changed")
	}
	if got.ExpectedNewRootEndBytes !=
		manifest().ExpectedNewRootEndBytes {
		t.Fatal("new root end changed")
	}
}

func TestTamperedPayloadRejected(t *testing.T) {
	pub, priv := keypair(t)

	envelope, err := Sign(manifest(), priv)
	if err != nil {
		t.Fatal(err)
	}

	var payload map[string]any
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		t.Fatal(err)
	}

	payload["recovery_bytes"] = float64(99)

	envelope.Payload, err = json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := Verify(envelope, pub); err == nil ||
		!strings.Contains(err.Error(), "signature") {
		t.Fatalf("error=%v", err)
	}
}

func TestWrongPublicKeyRejected(t *testing.T) {
	_, priv := keypair(t)
	wrongPub, _ := keypair(t)

	envelope, err := Sign(manifest(), priv)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := Verify(envelope, wrongPub); err == nil {
		t.Fatal("wrong trust anchor accepted")
	}
}

func TestUnknownPayloadFieldRejectedEvenWhenSigned(t *testing.T) {
	pub, priv := keypair(t)

	raw := []byte(`{
	  "version":1,
	  "nonce":"nonce",
	  "stage":"preparing-recovery-resize",
	  "disk_guid":"aaaaaaaa",
	  "disk_size_bytes":100000,
	  "root_partuuid":"bbbbbbbb",
	  "root_start_bytes":1000,
	  "expected_current_root_end_bytes":90000,
	  "expected_new_root_end_bytes":80000,
	  "luks_uuid":"cccccccc",
	  "btrfs_uuid":"dddddddd",
	  "recovery_bytes":10000,
	  "command":"rm -rf /"
	}`)

	envelope := Envelope{
		Version:   EnvelopeVersion,
		Payload:   raw,
		Signature: "",
	}
	envelope.Signature = encodeSignature(
		ed25519.Sign(priv, raw),
	)

	_, err := Verify(envelope, pub)
	if err == nil ||
		!strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("error=%v", err)
	}
}

func TestEnvelopeRejectsUnknownTopLevelField(t *testing.T) {
	raw := []byte(`{
	  "version":1,
	  "payload":{},
	  "signature":"abc",
	  "command":"anything"
	}`)

	_, err := UnmarshalEnvelope(raw)
	if err == nil ||
		!strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("error=%v", err)
	}
}

func TestManifestContainsNoCredentialFields(t *testing.T) {
	data, err := json.Marshal(manifest())
	if err != nil {
		t.Fatal(err)
	}

	lower := strings.ToLower(string(data))

	for _, forbidden := range []string{
		"passphrase",
		"password",
		"private_key",
		"keyfile",
		"unlock_secret",
		"command",
		"script",
	} {
		if strings.Contains(lower, forbidden) {
			t.Fatalf(
				"manifest contains forbidden field marker %q:\n%s",
				forbidden,
				data,
			)
		}
	}
}

func TestInvalidGeometryCannotBeSigned(t *testing.T) {
	_, priv := keypair(t)

	m := manifest()
	m.ExpectedNewRootEndBytes =
		m.ExpectedCurrentRootEndBytes

	_, err := Sign(m, priv)
	if err == nil {
		t.Fatal("invalid geometry was signed")
	}
}

func TestPublicKeyFingerprintStable(t *testing.T) {
	pub, _ := keypair(t)

	a, err := PublicKeyFingerprint(pub)
	if err != nil {
		t.Fatal(err)
	}

	b, err := PublicKeyFingerprint(pub)
	if err != nil {
		t.Fatal(err)
	}

	if a != b || len(a) != 64 {
		t.Fatalf("fingerprints %q %q", a, b)
	}
}

func encodeSignature(signature []byte) string {
	const table = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

	out := make([]byte, 0, ((len(signature)+2)/3)*4)

	for len(signature) >= 3 {
		v := uint32(signature[0])<<16 |
			uint32(signature[1])<<8 |
			uint32(signature[2])

		out = append(
			out,
			table[(v>>18)&63],
			table[(v>>12)&63],
			table[(v>>6)&63],
			table[v&63],
		)
		signature = signature[3:]
	}

	if len(signature) == 1 {
		v := uint32(signature[0]) << 16
		out = append(
			out,
			table[(v>>18)&63],
			table[(v>>12)&63],
			'=',
			'=',
		)
	} else if len(signature) == 2 {
		v := uint32(signature[0])<<16 |
			uint32(signature[1])<<8
		out = append(
			out,
			table[(v>>18)&63],
			table[(v>>12)&63],
			table[(v>>6)&63],
			'=',
		)
	}

	return string(out)
}
