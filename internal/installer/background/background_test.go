package background

import (
	"context"
	"path/filepath"
	"testing"
)

func TestResolveRelativeAndAbsolute(t *testing.T) {
	root := t.TempDir()
	dotfiles := filepath.Join(root, "repo")
	got, err := Resolve(context.Background(), root, dotfiles, "normal", "images/a.png")
	if err != nil || got != filepath.Join(dotfiles, "images/a.png") {
		t.Fatalf("got %q, %v", got, err)
	}
	abs := filepath.Join(root, "a.png")
	got, err = Resolve(context.Background(), root, dotfiles, "normal", abs)
	if err != nil || got != abs {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestResolveRejectsUnsafeURL(t *testing.T) {
	if _, err := Resolve(context.Background(), t.TempDir(), "/repo", "normal", "http://example.com/a.png"); err == nil {
		t.Fatal("accepted HTTP URL")
	}
}
