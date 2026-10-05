package recoveryresize

import (
	"context"
	"slices"
	"testing"
)

func TestCommandDropsSudoOnlyForRoot(t *testing.T) {
	previous := geteuid
	t.Cleanup(func() { geteuid = previous })

	readOnly := func() []string {
		name, args := privilegedReadOnlyCommand("cryptsetup", []string{"status", "cryptroot"})
		return command(context.Background(), name, args...).Args
	}

	geteuid = func() int { return 0 }
	if got := readOnly(); !slices.Equal(got, []string{"cryptsetup", "status", "cryptroot"}) {
		t.Fatalf("root read-only args = %v", got)
	}
	if got := command(context.Background(), "sudo", "sync").Args; !slices.Equal(got, []string{"sync"}) {
		t.Fatalf("root args = %v", got)
	}

	geteuid = func() int { return 1000 }
	if got := readOnly(); !slices.Equal(got, []string{"sudo", "-n", "--", "cryptsetup", "status", "cryptroot"}) {
		t.Fatalf("non-root read-only args = %v", got)
	}
	if got := command(context.Background(), "sudo", "sync").Args; !slices.Equal(got, []string{"sudo", "sync"}) {
		t.Fatalf("non-root args = %v", got)
	}
}
