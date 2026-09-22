package usbtrust

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	TrustFile     = "trust.json"
	SignatureFile = "trust.sig.json"
)

type Signature struct {
	Schema       int    `json:"schema"`
	Algorithm    string `json:"algorithm"`
	KeyReference string `json:"keyReference"`
	DigestSHA256 string `json:"digestSha256"`
	Value        string `json:"value"`
}

type Signer interface {
	Sign(context.Context, []byte) (Signature, error)
	Verify(context.Context, []byte, Signature) error
}

func WriteSigned(
	ctx context.Context,
	dir string,
	document Document,
	signer Signer,
) error {
	payload, err := Canonical(document)
	if err != nil {
		return fmt.Errorf("canonicalize USB trust document: %w", err)
	}

	signature, err := signer.Sign(ctx, payload)
	if err != nil {
		return fmt.Errorf("sign USB trust document: %w", err)
	}

	signatureData, err := json.Marshal(signature)
	if err != nil {
		return fmt.Errorf("encode USB trust signature: %w", err)
	}

	if err := EnsureSecureStateDirectory(dir); err != nil {
		return err
	}

	trustPath := filepath.Join(dir, TrustFile)

	if err := writeAtomic(
		trustPath,
		payload,
		StateFileMode,
	); err != nil {
		return err
	}

	if err := ValidateSecureStateFile(trustPath); err != nil {
		return err
	}

	signaturePath := filepath.Join(dir, SignatureFile)

	if err := writeAtomic(
		signaturePath,
		signatureData,
		StateFileMode,
	); err != nil {
		return err
	}

	if err := ValidateSecureStateFile(signaturePath); err != nil {
		return err
	}

	return nil
}

func ReadVerified(
	ctx context.Context,
	dir string,
	signer Signer,
) (Document, error) {
	if present, err := pathPresent(filepath.Join(dir, EnvelopeFile)); err != nil {
		return Document{}, err
	} else if present {
		return readEnvelope(ctx, dir, signer)
	}
	var document Document

	if err := EnsureSecureStateDirectory(dir); err != nil {
		return document, err
	}

	trustPath := filepath.Join(dir, TrustFile)
	signaturePath := filepath.Join(dir, SignatureFile)

	if err := ValidateSecureStateFile(trustPath); err != nil {
		return document, err
	}

	if err := ValidateSecureStateFile(signaturePath); err != nil {
		return document, err
	}

	documentData, err := os.ReadFile(trustPath)
	if err != nil {
		return document, fmt.Errorf("read USB trust document: %w", err)
	}

	signatureData, err := os.ReadFile(signaturePath)
	if err != nil {
		return document, fmt.Errorf("read USB trust signature: %w", err)
	}

	if err := json.Unmarshal(documentData, &document); err != nil {
		return document, fmt.Errorf("decode USB trust document: %w", err)
	}

	var signature Signature
	if err := json.Unmarshal(signatureData, &signature); err != nil {
		return document, fmt.Errorf("decode USB trust signature: %w", err)
	}

	payload, err := Canonical(document)
	if err != nil {
		return document, fmt.Errorf("validate USB trust document: %w", err)
	}

	if err := signer.Verify(ctx, payload, signature); err != nil {
		return document, fmt.Errorf("verify USB trust document: %w", err)
	}

	return document, nil
}

func writeAtomic(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)

	tmp, err := os.CreateTemp(dir, ".usbtrust-*")
	if err != nil {
		return fmt.Errorf("create temporary USB trust file: %w", err)
	}

	tmpPath := tmp.Name()

	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()

	if err := tmp.Chmod(mode); err != nil {
		return err
	}

	if _, err := tmp.Write(data); err != nil {
		return err
	}

	if err := tmp.Sync(); err != nil {
		return err
	}

	if err := tmp.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("replace USB trust file: %w", err)
	}

	return nil
}
