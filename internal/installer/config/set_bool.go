package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
)

// SetBool changes one top-level boolean in an existing user.config.json and
// leaves every other byte alone, so key order, indentation and the file mode
// survive. The result is parsed again and must differ from the original only
// in that key, otherwise nothing is written.
func SetBool(path, key string, value bool) error {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		path = resolved
	}
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("read user configuration: %w", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read user configuration: %w", err)
	}
	var want map[string]any
	if err := json.Unmarshal(data, &want); err != nil {
		return fmt.Errorf("parse user configuration: %w", err)
	}
	want[key] = value

	updated := setBoolText(data, key, value)
	var got map[string]any
	if updated == nil || json.Unmarshal(updated, &got) != nil || !reflect.DeepEqual(got, want) {
		return fmt.Errorf("cannot change %q in place in %s", key, path)
	}
	return replaceFile(path, updated, info.Mode().Perm())
}

// setBoolText rewrites an existing "key": true|false, or inserts the key as
// the first entry with the indentation of the entry that follows. It returns
// nil when the key appears more than once or holds a non-boolean.
func setBoolText(data []byte, key string, value bool) []byte {
	literal := []byte(strconv.FormatBool(value))
	quoted := regexp.QuoteMeta(strconv.Quote(key))
	existing := regexp.MustCompile(quoted + `(\s*:\s*)(true|false)\b`)
	switch matches := existing.FindAllSubmatchIndex(data, -1); len(matches) {
	case 0:
		if regexp.MustCompile(quoted + `\s*:`).Match(data) {
			return nil
		}
	case 1:
		m := matches[0]
		return append(append(append([]byte{}, data[:m[4]]...), literal...), data[m[5]:]...)
	default:
		return nil
	}

	open := regexp.MustCompile(`^\s*\{(\s*)`).FindSubmatchIndex(data)
	if open == nil {
		return nil
	}
	space := data[open[2]:open[3]]
	rest := data[open[3]:]
	entry := strconv.Quote(key) + ": " + string(literal)
	if len(rest) > 0 && rest[0] != '}' {
		entry += ","
	}
	out := append([]byte{}, data[:open[2]]...)
	out = append(out, space...)
	out = append(out, entry...)
	out = append(out, space...)
	return append(out, rest...)
}

func replaceFile(path string, data []byte, mode os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".user.config.json-*")
	if err != nil {
		return fmt.Errorf("create temporary user configuration: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)

	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return fmt.Errorf("set user configuration permissions: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write user configuration: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("sync user configuration: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close user configuration: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace user configuration: %w", err)
	}
	return nil
}
