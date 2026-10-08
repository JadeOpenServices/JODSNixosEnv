package oddc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// UpstreamRevision asks the git repository remote which commit ref names.
// "" asks for the default branch. It gives up after timeout.
func UpstreamRevision(remote, ref string, timeout time.Duration) (string, error) {
	if ref == "" {
		ref = "HEAD"
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "ls-remote", "--", remote, ref)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("ask ODDC upstream: %w", err)
	}

	// ls-remote matches ref against the tail of every name, tags too.
	rev := ""
	for _, line := range strings.Split(string(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && (fields[1] == ref || fields[1] == "refs/heads/"+ref) {
			rev = fields[0]
			break
		}
	}
	if len(rev) != 40 || strings.Trim(rev, "0123456789abcdef") != "" {
		return "", errors.New("ask ODDC upstream: answer is not a commit")
	}
	return rev, nil
}
