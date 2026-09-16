package oddc

import "testing"

func TestEmbeddedSourceMetadata(t *testing.T) {
	source := EmbeddedSource{
		Root:       "/tmp/oddc",
		Repository: "https://example.invalid/oddc",
		Revision:   "git:test",
		Integrity:  "sha256:test",
	}

	got := source.Metadata()

	if got.Kind != "embedded" {
		t.Fatalf("Kind=%q", got.Kind)
	}
	if got.Repository != source.Repository {
		t.Fatalf("Repository=%q", got.Repository)
	}
	if got.Revision != source.Revision {
		t.Fatalf("Revision=%q", got.Revision)
	}
	if got.Integrity != source.Integrity {
		t.Fatalf("Integrity=%q", got.Integrity)
	}
}
