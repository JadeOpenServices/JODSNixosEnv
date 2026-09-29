package secureboot

import (
	"context"
	"slices"
	"testing"
)

func TestCommandContextDropsSudoOnlyForRoot(t *testing.T) {
	previous := geteuid
	t.Cleanup(func() { geteuid = previous })

	geteuid = func() int { return 0 }
	if got := commandContext(context.Background(), "sudo", "test", "-f", "/x").Args; !slices.Equal(got, []string{"test", "-f", "/x"}) {
		t.Fatalf("root args = %v", got)
	}
	if got := commandContext(context.Background(), "sbctl", "verify").Args; !slices.Equal(got, []string{"sbctl", "verify"}) {
		t.Fatalf("plain command args = %v", got)
	}

	geteuid = func() int { return 1000 }
	if got := commandContext(context.Background(), "sudo", "test", "-f", "/x").Args; !slices.Equal(got, []string{"sudo", "test", "-f", "/x"}) {
		t.Fatalf("non-root args = %v", got)
	}
}
