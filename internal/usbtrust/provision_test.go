package usbtrust

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// Explicitly opt in: this test always uses a disposable software TPM, never
// /dev/tpm*. The production signing/provisioning commands run unchanged.
func TestTPMProvisionAndSignedPersistenceSimulator(t *testing.T) {
	if os.Getenv("GJALLAR_USBTRUST_TPM_TEST") != "1" {
		t.Skip("software TPM integration is opt-in")
	}
	root := t.TempDir()
	socket := filepath.Join(root, "tpm.sock")
	cmd := exec.Command("swtpm", "socket", "--tpm2", "--tpmstate", "dir="+root, "--server", "type=unixio,path="+socket, "--ctrl", "type=unixio,path="+socket+".ctrl", "--flags", "not-need-init,startup-clear")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _ = cmd.Wait() })
	for i := 0; i < 100; i++ {
		if _, err := os.Stat(socket); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Setenv("TPM2TOOLS_TCTI", "swtpm:path="+socket)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	dir := filepath.Join(root, "trust")
	const handle = "0x81000042"
	if err := ProvisionTPM(ctx, dir, handle); err != nil {
		t.Fatal(err)
	}
	if err := ProvisionTPM(ctx, dir, handle); err == nil {
		t.Fatal("occupied TPM handle replaced")
	}
	signer, _ := NewTPMSigner(handle)
	if err := CommitSigned(ctx, dir, sampleDocument(), signer); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadVerified(ctx, dir, signer); err != nil {
		t.Fatal(err)
	}
	other, _ := NewTPMSigner("0x81000043")
	if _, err := ReadVerified(ctx, dir, other); err == nil {
		t.Fatal("wrong key accepted")
	}
}
