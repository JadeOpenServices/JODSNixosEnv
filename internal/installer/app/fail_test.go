package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/JadeOpenServices/gjallarOS/internal/installer/credential"
)

func TestFailReportsInterruptAsStop(t *testing.T) {
	for _, tc := range []struct {
		err  error
		code int
		want string
	}{
		{context.Canceled, 130, "Installer stopped.\n"},
		{credential.ErrInterrupted, 130, "Installer stopped.\n"},
		{fmt.Errorf("unmount target: %w", context.Canceled), 130, "Installer stopped: unmount target: context canceled\n"},
		{errors.New("disk is gone"), 1, "ERROR: disk is gone\n"},
	} {
		var out bytes.Buffer
		if code := fail(&out, tc.err); code != tc.code || out.String() != tc.want {
			t.Errorf("fail(%v) = %d %q, want %d %q", tc.err, code, out.String(), tc.code, tc.want)
		}
	}
}
