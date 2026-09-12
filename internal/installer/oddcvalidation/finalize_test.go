package oddcvalidation

import "testing"

func TestDecideWritebackModesRemainStable(t *testing.T) {
	if got := DecideWriteback(Authority{}); got != WritebackLocalOnly {
		t.Fatalf("unauthorized mode = %q", got)
	}

	if got := DecideWriteback(Authority{Allowed: true}); got != WritebackUpstream {
		t.Fatalf("authorized mode = %q", got)
	}
}
