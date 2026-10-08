// Package resizebootstrap authenticates the non-secret recovery-resize
// operation state passed into a one-shot maintenance environment.
//
// It deliberately does not execute storage commands, manage boot entries,
// unlock disks, contain credentials, or trust geometry from the manifest.
// The maintenance environment must independently rediscover and validate the
// actual storage topology before any later executor is allowed to mutate it.
package resizebootstrap

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/JadeOpenServices/gjallarOS/internal/installer/recoveryresize"
)

const EnvelopeVersion = 1

type Envelope struct {
	Version   int             `json:"version"`
	Payload   json.RawMessage `json:"payload"`
	Signature string          `json:"signature"`
}

// Sign authenticates an already validated non-secret recovery-resize manifest.
//
// The private signing key is supplied by the caller and is not serialized into
// the envelope. The resulting signature covers the exact payload bytes.
func Sign(
	manifest recoveryresize.Manifest,
	privateKey ed25519.PrivateKey,
) (Envelope, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return Envelope{}, errors.New(
			"resize manifest Ed25519 private key has invalid length",
		)
	}

	if err := validateManifestShape(manifest); err != nil {
		return Envelope{}, err
	}

	payload, err := json.Marshal(manifest)
	if err != nil {
		return Envelope{}, fmt.Errorf(
			"serialize recovery resize manifest: %w",
			err,
		)
	}

	signature := ed25519.Sign(privateKey, payload)

	return Envelope{
		Version:   EnvelopeVersion,
		Payload:   append(json.RawMessage(nil), payload...),
		Signature: base64.StdEncoding.EncodeToString(signature),
	}, nil
}

func Verify(
	envelope Envelope,
	publicKey ed25519.PublicKey,
) (recoveryresize.Manifest, error) {
	if envelope.Version != EnvelopeVersion {
		return recoveryresize.Manifest{}, fmt.Errorf(
			"unsupported resize-bootstrap envelope version %d",
			envelope.Version,
		)
	}
	if len(publicKey) != ed25519.PublicKeySize {
		return recoveryresize.Manifest{}, errors.New(
			"resize manifest Ed25519 public key has invalid length",
		)
	}
	if len(envelope.Payload) == 0 {
		return recoveryresize.Manifest{}, errors.New(
			"resize manifest payload is empty",
		)
	}

	signature, err := base64.StdEncoding.DecodeString(
		strings.TrimSpace(envelope.Signature),
	)
	if err != nil {
		return recoveryresize.Manifest{}, fmt.Errorf(
			"decode resize manifest signature: %w",
			err,
		)
	}
	if len(signature) != ed25519.SignatureSize {
		return recoveryresize.Manifest{}, errors.New(
			"resize manifest signature has invalid length",
		)
	}

	if !ed25519.Verify(publicKey, envelope.Payload, signature) {
		return recoveryresize.Manifest{}, errors.New(
			"resize manifest signature verification failed",
		)
	}

	var manifest recoveryresize.Manifest
	decoder := json.NewDecoder(bytes.NewReader(envelope.Payload))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&manifest); err != nil {
		return recoveryresize.Manifest{}, fmt.Errorf(
			"decode authenticated resize manifest: %w",
			err,
		)
	}

	if err := requireJSONEOF(decoder); err != nil {
		return recoveryresize.Manifest{}, err
	}

	if err := validateManifestShape(manifest); err != nil {
		return recoveryresize.Manifest{}, err
	}

	return manifest, nil
}

func MarshalEnvelope(envelope Envelope) ([]byte, error) {
	if envelope.Version != EnvelopeVersion {
		return nil, fmt.Errorf(
			"unsupported envelope version %d",
			envelope.Version,
		)
	}
	if len(envelope.Payload) == 0 {
		return nil, errors.New("envelope payload is empty")
	}
	if strings.TrimSpace(envelope.Signature) == "" {
		return nil, errors.New("envelope signature is empty")
	}

	data, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf(
			"serialize resize bootstrap envelope: %w",
			err,
		)
	}
	return data, nil
}

func UnmarshalEnvelope(data []byte) (Envelope, error) {
	var envelope Envelope

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&envelope); err != nil {
		return Envelope{}, fmt.Errorf(
			"decode resize bootstrap envelope: %w",
			err,
		)
	}

	if err := requireJSONEOF(decoder); err != nil {
		return Envelope{}, err
	}

	if envelope.Version != EnvelopeVersion {
		return Envelope{}, fmt.Errorf(
			"unsupported resize-bootstrap envelope version %d",
			envelope.Version,
		)
	}
	if len(envelope.Payload) == 0 {
		return Envelope{}, errors.New(
			"resize bootstrap payload is empty",
		)
	}
	if strings.TrimSpace(envelope.Signature) == "" {
		return Envelope{}, errors.New(
			"resize bootstrap signature is empty",
		)
	}

	return envelope, nil
}

func PublicKeyFingerprint(publicKey ed25519.PublicKey) (string, error) {
	if len(publicKey) != ed25519.PublicKeySize {
		return "", errors.New(
			"resize manifest Ed25519 public key has invalid length",
		)
	}

	sum := sha256.Sum256(publicKey)
	return hex.EncodeToString(sum[:]), nil
}

func validateManifestShape(
	manifest recoveryresize.Manifest,
) error {
	if manifest.Version != recoveryresize.ManifestVersion {
		return fmt.Errorf(
			"unsupported recovery resize manifest version %d",
			manifest.Version,
		)
	}
	if strings.TrimSpace(manifest.Nonce) == "" {
		return errors.New("resize manifest nonce is required")
	}
	if strings.TrimSpace(manifest.Stage) == "" {
		return errors.New("resize manifest stage is required")
	}

	for name, value := range map[string]string{
		"disk GUID":     manifest.DiskGUID,
		"root PARTUUID": manifest.RootPARTUUID,
		"LUKS UUID":     manifest.LUKSUUID,
		"Btrfs UUID":    manifest.BtrfsUUID,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf(
				"resize manifest %s is required",
				name,
			)
		}
	}

	if manifest.DiskSizeBytes == 0 {
		return errors.New(
			"resize manifest disk size is required",
		)
	}
	if manifest.RootStartBytes == 0 {
		return errors.New(
			"resize manifest root start is required",
		)
	}
	if manifest.ExpectedCurrentRootEndBytes <=
		manifest.RootStartBytes {
		return errors.New(
			"resize manifest current root geometry is invalid",
		)
	}
	if manifest.ExpectedNewRootEndBytes <=
		manifest.RootStartBytes {
		return errors.New(
			"resize manifest new root geometry is invalid",
		)
	}
	if manifest.ExpectedNewRootEndBytes >=
		manifest.ExpectedCurrentRootEndBytes {
		return errors.New(
			"resize manifest does not reduce the root end",
		)
	}
	if manifest.RecoveryBytes == 0 {
		return errors.New(
			"resize manifest recovery size is required",
		)
	}

	return nil
}

func requireJSONEOF(decoder *json.Decoder) error {
	var extra any
	err := decoder.Decode(&extra)
	if errors.Is(err, io.EOF) {
		return nil
	}
	if err == nil {
		return errors.New(
			"unexpected trailing JSON value",
		)
	}
	return fmt.Errorf("parse trailing JSON: %w", err)
}
