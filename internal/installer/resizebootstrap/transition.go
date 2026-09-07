package resizebootstrap

import (
	"fmt"

	"github.com/bakanura/gjallarOS/internal/installer/recoveryresize"
)

// ValidateStageTransition prevents authenticated resume state from jumping
// directly over required validation or verification stages.
func ValidateStageTransition(current, next string) error {
	allowed := map[string]string{
		recoveryresize.StagePreparingResize: recoveryresize.StageValidatingEnvironment,

		recoveryresize.StageValidatingEnvironment: recoveryresize.StageResizingRootStorage,

		recoveryresize.StageResizingRootStorage: recoveryresize.StageCreatingRecovery,

		recoveryresize.StageCreatingRecovery: recoveryresize.StageVerifyingRecovery,

		recoveryresize.StageVerifyingRecovery: recoveryresize.StageRecoveryReady,
	}

	expected, ok := allowed[current]
	if !ok {
		return fmt.Errorf(
			"resume stage %q has no permitted successor",
			current,
		)
	}

	if next != expected {
		return fmt.Errorf(
			"invalid recovery resize stage transition: %q -> %q; expected %q",
			current,
			next,
			expected,
		)
	}

	return nil
}
