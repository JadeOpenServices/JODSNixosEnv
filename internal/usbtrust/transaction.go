package usbtrust

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const EnvelopeFile = "signed-trust.json"

type signedEnvelope struct {
	Document  Document  `json:"document"`
	Signature Signature `json:"signature"`
}

// CommitSigned replaces document and signature as a single durable unit.
// Legacy two-file state remains readable but is never written by the daemon.
func CommitSigned(ctx context.Context, dir string, document Document, signer Signer) error {
	if signer == nil {
		return fmt.Errorf("TPM signer is not configured")
	}
	payload, err := Canonical(document)
	if err != nil {
		return err
	}
	signature, err := signer.Sign(ctx, payload)
	if err != nil {
		return err
	}
	if err := signer.Verify(ctx, payload, signature); err != nil {
		return fmt.Errorf("verify new USB trust signature: %w", err)
	}
	data, err := json.Marshal(signedEnvelope{Document: document, Signature: signature})
	if err != nil {
		return err
	}
	if err := EnsureSecureStateDirectory(dir); err != nil {
		return err
	}
	path := filepath.Join(dir, EnvelopeFile)
	if present, err := pathPresent(path); err != nil {
		return err
	} else if present {
		if err := ValidateSecureStateFile(path); err != nil {
			return err
		}
	}
	if err := writeAtomic(path, data, StateFileMode); err != nil {
		return err
	}
	directory, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return err
	}
	// Once the envelope is durable, retire the old pair so normal migration
	// does not leave a second historical database beside the current state.
	for _, name := range []string{TrustFile, SignatureFile} {
		if err := os.Remove(filepath.Join(dir, name)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return directory.Sync()
}

func readEnvelope(ctx context.Context, dir string, signer Signer) (Document, error) {
	var envelope signedEnvelope
	if signer == nil {
		return Document{}, fmt.Errorf("TPM signer is not configured")
	}
	if err := EnsureSecureStateDirectory(dir); err != nil {
		return Document{}, err
	}
	path := filepath.Join(dir, EnvelopeFile)
	if err := ValidateSecureStateFile(path); err != nil {
		return Document{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Document{}, err
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return Document{}, err
	}
	payload, err := Canonical(envelope.Document)
	if err != nil {
		return Document{}, err
	}
	if err := signer.Verify(ctx, payload, envelope.Signature); err != nil {
		return Document{}, err
	}
	return envelope.Document, nil
}
