package prompt

import (
	"bufio"
	"context"
	"strings"
	"testing"
)

func TestTerminalDefaults(t *testing.T) {
	var out strings.Builder
	u := UI{Reader: bufio.NewReader(strings.NewReader("\n\n")), Out: &out}
	yes, err := u.Confirm(context.Background(), "Continue?", true)
	if err != nil || !yes {
		t.Fatal(yes, err)
	}
	value, err := u.Value(context.Background(), "Name", "default")
	if err != nil || value != "default" {
		t.Fatal(value, err)
	}
}
