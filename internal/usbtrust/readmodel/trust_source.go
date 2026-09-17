package readmodel

import (
	"context"
	"fmt"
	"strings"

	"github.com/bakanura/gjallarOS/internal/usbtrust"
)

type VerifiedReader func(
	ctx context.Context,
	directory string,
) (usbtrust.Document, error)

type VerifiedTrustSource struct {
	Directory string
	Verify    VerifiedReader
}

func NewVerifiedTrustSource(
	directory string,
	signer usbtrust.Signer,
) VerifiedTrustSource {
	return VerifiedTrustSource{
		Directory: directory,
		Verify: func(
			ctx context.Context,
			path string,
		) (usbtrust.Document, error) {
			if signer == nil {
				return usbtrust.Document{}, fmt.Errorf(
					"USB trust signer is unavailable",
				)
			}

			return usbtrust.ReadVerified(
				ctx,
				path,
				signer,
			)
		},
	}
}

func (s VerifiedTrustSource) Trusted(
	ctx context.Context,
) (*usbtrust.Document, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	directory := strings.TrimSpace(s.Directory)
	if directory == "" {
		return nil, fmt.Errorf(
			"USB trust state directory is not configured",
		)
	}

	present, err := usbtrust.SignedStatePresent(
		directory,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"inspect USB trust signed state: %w",
			err,
		)
	}

	if !present {
		return nil, nil
	}

	if s.Verify == nil {
		return nil, fmt.Errorf(
			"USB trust verifier is unavailable",
		)
	}

	document, err := s.Verify(
		ctx,
		directory,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"verify USB trust signed state: %w",
			err,
		)
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if err := document.Validate(); err != nil {
		return nil, fmt.Errorf(
			"verified USB trust document is invalid: %w",
			err,
		)
	}

	return &document, nil
}
