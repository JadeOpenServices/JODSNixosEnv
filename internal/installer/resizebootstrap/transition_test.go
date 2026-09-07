package resizebootstrap

import (
	"strings"
	"testing"

	"github.com/bakanura/gjallarOS/internal/installer/recoveryresize"
)

func TestValidStageTransitions(t *testing.T) {
	steps := [][2]string{
		{
			recoveryresize.StagePreparingResize,
			recoveryresize.StageValidatingEnvironment,
		},
		{
			recoveryresize.StageValidatingEnvironment,
			recoveryresize.StageResizingRootStorage,
		},
		{
			recoveryresize.StageResizingRootStorage,
			recoveryresize.StageCreatingRecovery,
		},
		{
			recoveryresize.StageCreatingRecovery,
			recoveryresize.StageVerifyingRecovery,
		},
		{
			recoveryresize.StageVerifyingRecovery,
			recoveryresize.StageRecoveryReady,
		},
	}

	for _, step := range steps {
		if err := ValidateStageTransition(
			step[0],
			step[1],
		); err != nil {
			t.Fatalf("%q -> %q: %v", step[0], step[1], err)
		}
	}
}

func TestCannotSkipValidationStage(t *testing.T) {
	err := ValidateStageTransition(
		recoveryresize.StagePreparingResize,
		recoveryresize.StageResizingRootStorage,
	)
	if err == nil ||
		!strings.Contains(err.Error(), "invalid") {
		t.Fatalf("error=%v", err)
	}
}

func TestReadyStateHasNoSuccessor(t *testing.T) {
	err := ValidateStageTransition(
		recoveryresize.StageRecoveryReady,
		recoveryresize.StageRecoveryReady,
	)
	if err == nil ||
		!strings.Contains(err.Error(), "no permitted successor") {
		t.Fatalf("error=%v", err)
	}
}
