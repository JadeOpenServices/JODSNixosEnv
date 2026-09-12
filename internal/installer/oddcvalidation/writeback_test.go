package oddcvalidation

import "testing"

func TestDecideWriteback(t *testing.T) {
	if got := DecideWriteback(Authority{}); got != WritebackLocalOnly {
		t.Fatalf("got %q", got)
	}

	if got := DecideWriteback(Authority{Allowed: true}); got != WritebackUpstream {
		t.Fatalf("got %q", got)
	}
}
