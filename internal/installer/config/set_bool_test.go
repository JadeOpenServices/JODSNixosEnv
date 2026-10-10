package config

import (
	"os"
	"path/filepath"
	"testing"
)

func writeConfigFixture(t *testing.T, body string, mode os.FileMode) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "user.config.json")
	if err := os.WriteFile(path, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
	return path
}

func readConfigFixture(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestSetBoolChangesOnlyThatKey(t *testing.T) {
	body := "{\n    \"username\": \"jo\",\n    \"usbReviewTechnicalView\": false,\n    \"hostname\": \"box\"\n}\n"
	path := writeConfigFixture(t, body, 0o640)

	if err := SetBool(path, "usbReviewTechnicalView", true); err != nil {
		t.Fatal(err)
	}
	want := "{\n    \"username\": \"jo\",\n    \"usbReviewTechnicalView\": true,\n    \"hostname\": \"box\"\n}\n"
	if got := readConfigFixture(t, path); got != want {
		t.Fatalf("file =\n%s\nwant\n%s", got, want)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("mode = %o, want 640", info.Mode().Perm())
	}

	if err := SetBool(path, "usbReviewTechnicalView", false); err != nil {
		t.Fatal(err)
	}
	if got := readConfigFixture(t, path); got != body {
		t.Fatalf("file =\n%s\nwant\n%s", got, body)
	}
}

func TestSetBoolInsertsMissingKeyFirst(t *testing.T) {
	path := writeConfigFixture(t, "{\n  \"hostname\": \"box\"\n}\n", 0o600)

	if err := SetBool(path, "usbReviewTechnicalView", true); err != nil {
		t.Fatal(err)
	}
	want := "{\n  \"usbReviewTechnicalView\": true,\n  \"hostname\": \"box\"\n}\n"
	if got := readConfigFixture(t, path); got != want {
		t.Fatalf("file =\n%s\nwant\n%s", got, want)
	}
}

func TestSetBoolRefusesWhatItCannotEditSafely(t *testing.T) {
	for name, body := range map[string]string{
		"not a boolean": "{\n  \"usbReviewTechnicalView\": \"yes\"\n}\n",
		"nested copy":   "{\n  \"usbReviewTechnicalView\": false,\n  \"apps\": {\"usbReviewTechnicalView\": false}\n}\n",
		"broken json":   "{\n  \"usbReviewTechnicalView\": false,\n",
	} {
		t.Run(name, func(t *testing.T) {
			path := writeConfigFixture(t, body, 0o600)
			if err := SetBool(path, "usbReviewTechnicalView", true); err == nil {
				t.Fatal("SetBool succeeded")
			}
			if got := readConfigFixture(t, path); got != body {
				t.Fatalf("file changed to\n%s", got)
			}
		})
	}
}
