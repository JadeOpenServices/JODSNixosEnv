// Package preset reads the flat, typed installer preset format.
package preset

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

type Document map[string]json.RawMessage

func Load(path string) (Document, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read preset: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(contents))
	var document Document
	if err := decoder.Decode(&document); err != nil || document == nil {
		if err == nil {
			err = fmt.Errorf("top-level value must be an object")
		}
		return nil, fmt.Errorf("parse preset: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, fmt.Errorf("parse preset: trailing JSON value")
	}
	return document, nil
}

func (d Document) String(key string) (string, error) {
	raw, ok := d[key]
	if !ok {
		return "", nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", fmt.Errorf("preset field %q is not a string", key)
	}
	if strings.ContainsAny(value, "\x00\r\n") {
		return "", fmt.Errorf("preset field %q contains control characters", key)
	}
	return value, nil
}

func (d Document) Bool(key string) (bool, error) {
	raw, ok := d[key]
	if !ok {
		return false, nil
	}
	var value bool
	if err := json.Unmarshal(raw, &value); err != nil {
		return false, fmt.Errorf("preset field %q is not a boolean", key)
	}
	return value, nil
}

func (d Document) Strings(key string) ([]string, error) {
	raw, ok := d[key]
	if !ok {
		return nil, nil
	}
	var values []string
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil, fmt.Errorf("preset field %q is not a string list", key)
	}
	for _, value := range values {
		if strings.ContainsAny(value, "\x00\r\n") {
			return nil, fmt.Errorf("preset field %q contains control characters", key)
		}
	}
	return values, nil
}
