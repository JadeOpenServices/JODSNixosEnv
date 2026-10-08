package usbtrust

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"strings"
	"testing"
)

func TestParseTPMPublicKeyPEMAcceptsP256(
	t *testing.T,
) {
	privateKey, err := ecdsa.GenerateKey(
		elliptic.P256(),
		rand.Reader,
	)
	if err != nil {
		t.Fatal(err)
	}

	got, err := parseTPMPublicKeyPEM(
		publicKeyPEM(t, &privateKey.PublicKey),
	)
	if err != nil {
		t.Fatal(err)
	}

	if got.Curve.Params().Name !=
		elliptic.P256().Params().Name {
		t.Fatalf(
			"curve = %q",
			got.Curve.Params().Name,
		)
	}
}

func TestParseTPMPublicKeyPEMRejectsP384(
	t *testing.T,
) {
	privateKey, err := ecdsa.GenerateKey(
		elliptic.P384(),
		rand.Reader,
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = parseTPMPublicKeyPEM(
		publicKeyPEM(t, &privateKey.PublicKey),
	)
	if err == nil {
		t.Fatal("P-384 key accepted as P-256")
	}

	if !strings.Contains(
		err.Error(),
		"require P-256",
	) {
		t.Fatalf(
			"unexpected error %q",
			err,
		)
	}
}

func TestParseTPMPublicKeyPEMRejectsRSA(
	t *testing.T,
) {
	privateKey, err := rsa.GenerateKey(
		rand.Reader,
		2048,
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = parseTPMPublicKeyPEM(
		publicKeyPEM(t, &privateKey.PublicKey),
	)
	if err == nil {
		t.Fatal("RSA key accepted as ECDSA P-256")
	}
}

func TestTPMSignerVerifyUsesP256PublicKey(
	t *testing.T,
) {
	const handle = "0x81000042"

	privateKey, err := ecdsa.GenerateKey(
		elliptic.P256(),
		rand.Reader,
	)
	if err != nil {
		t.Fatal(err)
	}

	payload := []byte(
		"synthetic USB trust document",
	)

	digest := sha256.Sum256(payload)

	rawSignature, err := ecdsa.SignASN1(
		rand.Reader,
		privateKey,
		digest[:],
	)
	if err != nil {
		t.Fatal(err)
	}

	signer := &TPMSigner{
		Handle: handle,
		readPublic: func(
			_ context.Context,
			gotHandle string,
		) ([]byte, error) {
			if gotHandle != handle {
				t.Fatalf(
					"handle = %q, want %q",
					gotHandle,
					handle,
				)
			}

			return publicKeyPEM(
				t,
				&privateKey.PublicKey,
			), nil
		},
	}

	err = signer.Verify(
		context.Background(),
		payload,
		Signature{
			Schema:       1,
			Algorithm:    TPMAlgorithm,
			KeyReference: handle,
			DigestSHA256: hex.EncodeToString(
				digest[:],
			),
			Value: base64.StdEncoding.EncodeToString(
				rawSignature,
			),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
}

func TestTPMSignerVerifyRejectsWrongSignature(
	t *testing.T,
) {
	const handle = "0x81000042"

	privateKey, err := ecdsa.GenerateKey(
		elliptic.P256(),
		rand.Reader,
	)
	if err != nil {
		t.Fatal(err)
	}

	payload := []byte("payload")
	digest := sha256.Sum256(payload)

	signer := &TPMSigner{
		Handle: handle,
		readPublic: func(
			context.Context,
			string,
		) ([]byte, error) {
			return publicKeyPEM(
				t,
				&privateKey.PublicKey,
			), nil
		},
	}

	err = signer.Verify(
		context.Background(),
		payload,
		Signature{
			Schema:       1,
			Algorithm:    TPMAlgorithm,
			KeyReference: handle,
			DigestSHA256: hex.EncodeToString(
				digest[:],
			),
			Value: base64.StdEncoding.EncodeToString(
				[]byte("not-a-valid-ecdsa-signature"),
			),
		},
	)
	if err == nil {
		t.Fatal("invalid signature verified")
	}
}

func TestTPMSignerVerifyRejectsWrongKeyType(
	t *testing.T,
) {
	const handle = "0x81000042"

	privateKey, err := rsa.GenerateKey(
		rand.Reader,
		2048,
	)
	if err != nil {
		t.Fatal(err)
	}

	payload := []byte("payload")
	digest := sha256.Sum256(payload)

	signer := &TPMSigner{
		Handle: handle,
		readPublic: func(
			context.Context,
			string,
		) ([]byte, error) {
			return publicKeyPEM(
				t,
				&privateKey.PublicKey,
			), nil
		},
	}

	err = signer.Verify(
		context.Background(),
		payload,
		Signature{
			Schema:       1,
			Algorithm:    TPMAlgorithm,
			KeyReference: handle,
			DigestSHA256: hex.EncodeToString(
				digest[:],
			),
			Value: base64.StdEncoding.EncodeToString(
				[]byte("synthetic"),
			),
		},
	)
	if err == nil {
		t.Fatal("wrong TPM key type accepted")
	}
}

func TestNewTPMSignerRejectsInvalidHandle(
	t *testing.T,
) {
	for _, handle := range []string{
		"not-a-handle",
		"0x80000000",
		"0x81100000",
		"0x81ffffff",
	} {
		if _, err := NewTPMSigner(handle); err == nil {
			t.Fatalf("invalid or unreserved TPM handle %q accepted", handle)
		}
	}

	if _, err := NewTPMSigner("0x81000042"); err != nil {
		t.Fatalf("reserved owner TPM handle rejected: %v", err)
	}
}

func publicKeyPEM(
	t *testing.T,
	publicKey any,
) []byte {
	t.Helper()

	der, err := x509.MarshalPKIXPublicKey(
		publicKey,
	)
	if err != nil {
		t.Fatal(err)
	}

	return pem.EncodeToMemory(
		&pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: der,
		},
	)
}

func TestTPMSignerReadsPublicKeyOnce(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	reads := 0
	signer := &TPMSigner{
		Handle: "0x81000042",
		readPublic: func(context.Context, string) ([]byte, error) {
			reads++
			return publicKeyPEM(t, &privateKey.PublicKey), nil
		},
	}
	for range 3 {
		if _, err := signer.publicKey(context.Background()); err != nil {
			t.Fatal(err)
		}
	}
	if reads != 1 {
		t.Fatalf("read the TPM public key %d times, want 1", reads)
	}
}

func TestTPMSignerRetriesFailedPublicKeyRead(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	reads := 0
	signer := &TPMSigner{
		Handle: "0x81000042",
		readPublic: func(context.Context, string) ([]byte, error) {
			reads++
			if reads == 1 {
				return nil, context.DeadlineExceeded
			}
			return publicKeyPEM(t, &privateKey.PublicKey), nil
		},
	}
	if _, err := signer.publicKey(context.Background()); err == nil {
		t.Fatal("first failed read was not reported")
	}
	if _, err := signer.publicKey(context.Background()); err != nil {
		t.Fatalf("failed read was cached: %v", err)
	}
}
