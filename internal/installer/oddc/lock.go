package oddc

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// LockedRevision is the ODDC commit repo's flake.lock pins. The machine's
// answer comes from that commit, so module code and catalog data match.
func LockedRevision(repo string) (string, error) {
	path := filepath.Join(repo, "flake.lock")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	var lock struct {
		Root  string `json:"root"`
		Nodes map[string]struct {
			Inputs map[string]json.RawMessage `json:"inputs"`
			Locked struct {
				Rev string `json:"rev"`
			} `json:"locked"`
		} `json:"nodes"`
	}
	if err := json.Unmarshal(data, &lock); err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}

	// A root input names its node; only follows (lists) point elsewhere.
	var node string
	if err := json.Unmarshal(lock.Nodes[lock.Root].Inputs["oddc"], &node); err != nil {
		return "", fmt.Errorf("%s has no oddc input", path)
	}

	rev := lock.Nodes[node].Locked.Rev
	if rev == "" {
		return "", fmt.Errorf("%s pins no oddc commit", path)
	}

	return rev, nil
}
