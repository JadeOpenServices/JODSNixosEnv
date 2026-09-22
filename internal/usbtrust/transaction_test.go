package usbtrust

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestAtomicSignedStateAndTampering(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state")
	signer := newTestSigner(t)
	ctx := context.Background()
	doc := sampleDocument()
	if err := CommitSigned(ctx, dir, doc, signer); err != nil {
		t.Fatal(err)
	}
	got, err := ReadVerified(ctx, dir, signer)
	if err != nil || got.Revision != 1 {
		t.Fatalf("read: %v %v", got, err)
	}
	if present, err := SignedStatePresent(dir); err != nil || !present {
		t.Fatalf("presence %t %v", present, err)
	}
	if err := CommitSigned(ctx, dir, doc, nil); err == nil {
		t.Fatal("unsigned commit accepted")
	}
	if _, err := ReadVerified(ctx, dir, signer); err != nil {
		t.Fatal("failed commit destroyed old state", err)
	}
	if err := os.WriteFile(filepath.Join(dir, EnvelopeFile), []byte(`{}`), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadVerified(ctx, dir, signer); err == nil {
		t.Fatal("tampered envelope accepted")
	}
}

func TestLegacyMigrationRetiresOldPair(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state")
	signer := newTestSigner(t)
	ctx := context.Background()
	doc := sampleDocument()
	if err := WriteSigned(ctx, dir, doc, signer); err != nil {
		t.Fatal(err)
	}
	doc.Revision++
	if err := CommitSigned(ctx, dir, doc, signer); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{TrustFile, SignatureFile} {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Fatalf("legacy %s remains: %v", name, err)
		}
	}
	got, err := ReadVerified(ctx, dir, signer)
	if err != nil || got.Revision != 2 {
		t.Fatalf("migration: %v %v", got, err)
	}
}
