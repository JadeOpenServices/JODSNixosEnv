package app

import (
	"os"
	"strings"
	"testing"
)

func TestTargetDiskSelectionContract(t *testing.T) {
	source, err := os.ReadFile("target_disk_selection.go")
	if err != nil {
		t.Fatal(err)
	}

	body := string(source)

	for _, want := range []string{
		"targetdisk.Discover(",
		"case 0:",
		"case 1:",
		"Automatically selected the only eligible target disk:",
		"Multiple eligible installation target disks detected:",
		"ui.Choice(",
		"Select the whole disk to erase and install GjallarOS onto",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("dynamic target selection contract missing %q", want)
		}
	}
}

func TestTargetDiskSelectionRetainsExplicitOverride(t *testing.T) {
	source, err := os.ReadFile("target_disk_selection.go")
	if err != nil {
		t.Fatal(err)
	}

	body := string(source)

	if !strings.Contains(
		body,
		"Using explicitly configured target disk:",
	) {
		t.Fatal("--target-disk explicit override was removed")
	}
}
