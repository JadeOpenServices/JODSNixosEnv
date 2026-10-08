package oddcvalidation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/JadeOpenServices/gjallarOS/internal/installer/oddc"
)

func LocalValidationMatches(
	ctx context.Context,
	target oddc.ValidationTarget,
) (bool, error) {
	data, err := defaultCommandRunner(
		ctx,
		"",
		"sudo",
		"cat",
		LocalValidationPath,
	)
	if err != nil {
		return false, nil
	}

	var record LocalValidationRecord
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&record); err != nil {
		return false, fmt.Errorf("decode local ODDC validation record: %w", err)
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return false, fmt.Errorf(
				"decode local ODDC validation record: trailing JSON value",
			)
		}
		return false, fmt.Errorf(
			"decode local ODDC validation record trailing data: %w",
			err,
		)
	}
	if record.Schema != 1 {
		return false, fmt.Errorf(
			"unsupported local ODDC validation schema %d",
			record.Schema,
		)
	}

	return record.Validation.Matches(target), nil
}
