package recoveryresize

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

type fakeOutputRunner struct {
	calls [][]string
	out   []byte
	err   error
}

func (f *fakeOutputRunner) Output(
	_ context.Context,
	name string,
	args ...string,
) ([]byte, error) {
	call := append([]string{name}, args...)
	f.calls = append(f.calls, call)

	if f.err != nil {
		return nil, f.err
	}
	return f.out, nil
}

func TestDetectActualBtrfsFilesystem(t *testing.T) {
	r := &fakeOutputRunner{
		out: []byte("btrfs\n"),
	}

	fs, err := DetectRootFilesystem(
		context.Background(),
		r,
		"/mnt",
	)
	if err != nil {
		t.Fatal(err)
	}
	if fs != "btrfs" {
		t.Fatalf("fs=%q", fs)
	}

	if len(r.calls) != 1 {
		t.Fatalf("calls=%v", r.calls)
	}

	joined := strings.Join(r.calls[0], " ")
	if joined != "findmnt -nro FSTYPE --target /mnt" {
		t.Fatalf("unexpected discovery call %q", joined)
	}
}

func TestActualExt4CapabilityGateFails(t *testing.T) {
	r := &fakeOutputRunner{
		out: []byte("ext4\n"),
	}

	fs, err := DetectRootFilesystem(
		context.Background(),
		r,
		"/mnt",
	)
	if err != nil {
		t.Fatal(err)
	}

	err = RequireBtrfsFilesystem(fs)
	if err == nil {
		t.Fatal("ext4 passed Btrfs capability gate")
	}
	if !strings.Contains(
		err.Error(),
		"Current filesystem: ext4. No disk changes were made.",
	) {
		t.Fatalf("unexpected UX: %v", err)
	}
}

func TestFilesystemDiscoveryFailureFailsClosed(t *testing.T) {
	r := &fakeOutputRunner{
		err: fmt.Errorf("findmnt failed"),
	}

	_, err := DetectRootFilesystem(
		context.Background(),
		r,
		"/mnt",
	)
	if err == nil {
		t.Fatal("discovery failure was ignored")
	}
}
