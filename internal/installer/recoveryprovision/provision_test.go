package recoveryprovision

import (
	"os"
	"strings"
	"testing"
)

func TestCanonicalRecoverySizeIsExactly12GiB(t *testing.T) {
	want := uint64(12 * 1024 * 1024 * 1024)

	if RecoveryBytes != want {
		t.Fatalf(
			"recovery size=%d want=%d",
			RecoveryBytes,
			want,
		)
	}
}

func TestRecoveryPlanningHasExplicitSafetyBudgets(t *testing.T) {
	if GPTSafetyMarginBytes == 0 {
		t.Fatal("GPT safety margin is zero")
	}
	if FilesystemHeadroomBytes == 0 {
		t.Fatal("Btrfs filesystem headroom is zero")
	}
	if AlignmentBytes == 0 {
		t.Fatal("partition alignment is zero")
	}
}

func TestPrepareValidatesTopologyBeforeGPTProvisioning(t *testing.T) {
	data, err := os.ReadFile("provision.go")
	if err != nil {
		t.Fatal(err)
	}

	text := string(data)

	discovery := strings.Index(
		text,
		"recoveryresize.DiscoverTopology(",
	)
	gpt := strings.Index(
		text,
		"gptprovision.ProvisionWithRunner(",
	)

	if discovery < 0 {
		t.Fatal("Prepare does not discover root topology")
	}
	if gpt < 0 {
		t.Fatal("Prepare does not invoke GPT provisioning")
	}
	if discovery >= gpt {
		t.Fatal(
			"GPT provisioning occurs before Btrfs topology validation",
		)
	}
}

func TestRecoveryProvisioningContainsNoWholeDiskZap(t *testing.T) {
	data, err := os.ReadFile("provision.go")
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(data), "--zap-all") {
		t.Fatal("recovery provisioning contains whole-disk wipe")
	}
}
