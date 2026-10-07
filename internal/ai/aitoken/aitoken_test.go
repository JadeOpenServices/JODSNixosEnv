package aitoken

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidate(t *testing.T) {
	good := strings.Repeat("a1", 32)
	if err := Validate(good); err != nil {
		t.Fatalf("hex token refused: %v", err)
	}
	for name, token := range map[string]string{
		"empty":   "",
		"short":   "abc123",
		"space":   good + " x",
		"newline": good + "\nx",
		"quote":   good + `"`,
		"long":    strings.Repeat("a", maxLength+1),
	} {
		if err := Validate(token); err == nil {
			t.Fatalf("%s: token accepted", name)
		}
	}
}

func TestReadTrimsAndValidates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token")
	good := strings.Repeat("b2", 32)
	if err := os.WriteFile(path, []byte(good+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := Read(path); err != nil || got != good {
		t.Fatalf("Read = %q, %v", got, err)
	}
	if err := os.WriteFile(path, []byte("short\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(path); err == nil {
		t.Fatal("short token accepted")
	}
}

func TestStoreKeepsTokenOutOfArgv(t *testing.T) {
	for _, args := range storeCommands(Pending) {
		for _, arg := range args {
			if strings.Contains(arg, "token") && arg != Pending {
				t.Fatalf("unexpected argument %q", arg)
			}
		}
	}
	last := storeCommands(Pending)[1]
	if last[len(last)-2] != "/dev/stdin" || last[len(last)-1] != Pending {
		t.Fatalf("store does not read stdin: %v", last)
	}
}
