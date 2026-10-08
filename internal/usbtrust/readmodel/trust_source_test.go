package readmodel

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JadeOpenServices/gjallarOS/internal/usbtrust"
)

func TestVerifiedTrustSourceMissingStateIsUnenrolled(
	t *testing.T,
) {
	source := VerifiedTrustSource{
		Directory: filepath.Join(
			t.TempDir(),
			"never-created",
		),
		Verify: func(
			context.Context,
			string,
		) (usbtrust.Document, error) {
			t.Fatal(
				"verifier called for absent state",
			)
			return usbtrust.Document{}, nil
		},
	}

	document, err := source.Trusted(
		context.Background(),
	)
	if err != nil {
		t.Fatal(err)
	}

	if document != nil {
		t.Fatal(
			"absent state became enrolled state",
		)
	}
}

func TestVerifiedTrustSourceIncompleteStateFailsClosed(
	t *testing.T,
) {
	dir := t.TempDir()

	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(
		filepath.Join(
			dir,
			usbtrust.TrustFile,
		),
		[]byte(`{}`),
		0600,
	); err != nil {
		t.Fatal(err)
	}

	source := VerifiedTrustSource{
		Directory: dir,
		Verify: func(
			context.Context,
			string,
		) (usbtrust.Document, error) {
			t.Fatal(
				"verifier called for incomplete state",
			)
			return usbtrust.Document{}, nil
		},
	}

	if _, err := source.Trusted(
		context.Background(),
	); err == nil {
		t.Fatal(
			"incomplete signed state became unenrolled",
		)
	}
}

func TestVerifiedTrustSourceUsesVerifiedDocument(
	t *testing.T,
) {
	dir := completeSyntheticState(t)

	calls := 0

	source := VerifiedTrustSource{
		Directory: dir,
		Verify: func(
			_ context.Context,
			path string,
		) (usbtrust.Document, error) {
			calls++

			if path != dir {
				t.Fatalf(
					"verify path = %q, want %q",
					path,
					dir,
				)
			}

			return validSyntheticDocument(), nil
		},
	}

	document, err := source.Trusted(
		context.Background(),
	)
	if err != nil {
		t.Fatal(err)
	}

	if document == nil {
		t.Fatal("verified state became nil")
	}

	if calls != 1 {
		t.Fatalf(
			"verify calls = %d, want 1",
			calls,
		)
	}

	if document.Revision != 3 {
		t.Fatalf(
			"revision = %d, want 3",
			document.Revision,
		)
	}
}

func TestVerifiedTrustSourceVerificationFailureFailsClosed(
	t *testing.T,
) {
	dir := completeSyntheticState(t)

	source := VerifiedTrustSource{
		Directory: dir,
		Verify: func(
			context.Context,
			string,
		) (usbtrust.Document, error) {
			return usbtrust.Document{},
				errors.New("synthetic signature failure")
		},
	}

	_, err := source.Trusted(
		context.Background(),
	)
	if err == nil {
		t.Fatal(
			"signature failure became trusted state",
		)
	}

	if !strings.Contains(
		err.Error(),
		"synthetic signature failure",
	) {
		t.Fatalf(
			"unexpected error %q",
			err,
		)
	}
}

func TestVerifiedTrustSourceRejectsInvalidVerifiedDocument(
	t *testing.T,
) {
	dir := completeSyntheticState(t)

	source := VerifiedTrustSource{
		Directory: dir,
		Verify: func(
			context.Context,
			string,
		) (usbtrust.Document, error) {
			document := validSyntheticDocument()
			document.Revision = 0
			return document, nil
		},
	}

	if _, err := source.Trusted(
		context.Background(),
	); err == nil {
		t.Fatal(
			"invalid verified document became trusted",
		)
	}
}

func TestVerifiedTrustSourceHonorsCancellation(
	t *testing.T,
) {
	ctx, cancel := context.WithCancel(
		context.Background(),
	)
	cancel()

	source := VerifiedTrustSource{
		Directory: t.TempDir(),
	}

	if _, err := source.Trusted(ctx); !errors.Is(
		err,
		context.Canceled,
	) {
		t.Fatalf(
			"cancellation error = %v",
			err,
		)
	}
}

func completeSyntheticState(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()

	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{
		usbtrust.TrustFile,
		usbtrust.SignatureFile,
	} {
		if err := os.WriteFile(
			filepath.Join(dir, name),
			[]byte(`synthetic`),
			0600,
		); err != nil {
			t.Fatal(err)
		}
	}

	return dir
}

func validSyntheticDocument() usbtrust.Document {
	return usbtrust.Document{
		Schema:    usbtrust.SchemaVersion,
		MachineID: "synthetic-machine",
		ODDCModel: "model/synthetic",
		Revision:  3,
		Devices:   nil,
	}
}
