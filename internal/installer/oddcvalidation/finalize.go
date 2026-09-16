package oddcvalidation

import (
	"context"

	"github.com/bakanura/gjallarOS/internal/installer/oddc"
)

func FinalizeValidation(
	ctx context.Context,
	repo string,
	modelID string,
	validation oddc.Validation,
	authority Authority,
) (WritebackMode, error) {
	if err := WriteLocalValidation(
		ctx,
		validation,
	); err != nil {
		return WritebackLocalOnly, err
	}

	if DecideWriteback(authority) !=
		WritebackUpstream {
		return WritebackLocalOnly, nil
	}

	if err := WriteValidationEvidence(
		repo,
		modelID,
		validation,
	); err != nil {
		return WritebackLocalOnly, err
	}

	return WritebackUpstream, nil
}
