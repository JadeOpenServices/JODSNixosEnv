package readmodel

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/JadeOpenServices/gjallarOS/internal/usbtrust/oddcsource"
)

func TestResolvedFilePreservesHostExpectations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resolved.json")
	if err := os.WriteFile(path, []byte(`{"hardware":{"input":{"local":{"bus":"usb","attachment":"internal","deviceId":"1234:5678"}}}}`), 0600); err != nil {
		t.Fatal(err)
	}
	resolved, err := ResolvedFile(path).Resolved(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	expected, err := oddcsource.Expected(resolved)
	if err != nil || len(expected) != 1 || expected[0].Role != "hardware.input.local" {
		t.Fatalf("%v %v", expected, err)
	}
}
