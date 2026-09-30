package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLUKSVerifierPassesKeyOnStdinOnly(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "cryptsetup")
	// The fake accepts one device and one key, read without a trailing newline.
	script := "#!/bin/sh\n[ \"$*\" = \"open --test-passphrase --disable-locks --key-file=- /dev/good\" ] || exit 1\n[ \"$(cat; echo x)\" = \"secretx\" ]\n"
	if err := os.WriteFile(fake, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	verify := luksVerifier(fake, []string{"/dev/other", "/dev/good"})
	if err := verify(context.Background(), []byte("secret")); err != nil {
		t.Fatal(err)
	}
	if err := verify(context.Background(), []byte("wrong")); err == nil {
		t.Fatal("wrong passphrase accepted")
	}
	if err := luksVerifier(fake, nil)(context.Background(), []byte("secret")); err == nil {
		t.Fatal("accepted with no LUKS devices")
	}
}
