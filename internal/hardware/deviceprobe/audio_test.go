package deviceprobe

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAudioDevices(t *testing.T) {
	root := t.TempDir()

	pci := filepath.Join(root, "bus/pci/devices/0000:04:00.6")
	driver := filepath.Join(root, "bus/pci/drivers/snd_hda_intel")
	cardTarget := filepath.Join(root, "devices/pci0000:00/0000:04:00.6/sound/card0")
	card := filepath.Join(root, "class/sound/card0")

	for _, dir := range []string{
		pci,
		driver,
		cardTarget,
		filepath.Dir(card),
	} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	if err := os.Symlink(cardTarget, card); err != nil {
		t.Fatal(err)
	}

	writeProbeFile(t, filepath.Join(cardTarget, "id"), "Generic")

	if err := os.Symlink(
		pci,
		filepath.Join(cardTarget, "device"),
	); err != nil {
		t.Fatal(err)
	}

	if err := os.Symlink(
		driver,
		filepath.Join(pci, "driver"),
	); err != nil {
		t.Fatal(err)
	}

	got := audioDevices(root)
	if len(got) != 1 {
		t.Fatalf("got %d audio devices, want 1", len(got))
	}

	if got[0].Card != "card0" ||
		got[0].ID != "Generic" ||
		got[0].Driver != "snd_hda_intel" ||
		got[0].DevicePath != "0000:04:00.6" {
		t.Fatalf("unexpected audio device: %+v", got[0])
	}
}
