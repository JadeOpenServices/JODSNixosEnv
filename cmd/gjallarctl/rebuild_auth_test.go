package main

import (
	"os"
	"os/exec"
	"testing"
)

func TestAttachControllingTTYBindsAllStreams(t *testing.T) {
	tty, err := os.CreateTemp(t.TempDir(), "tty")
	if err != nil {
		t.Fatal(err)
	}
	defer tty.Close()

	cmd := &exec.Cmd{}
	attachControllingTTY(cmd, tty)

	if cmd.Stdin != tty {
		t.Fatal("interactive authentication stdin is not the controlling TTY")
	}
	if cmd.Stdout != tty {
		t.Fatal("interactive authentication stdout is not the controlling TTY")
	}
	if cmd.Stderr != tty {
		t.Fatal("interactive authentication stderr is not the controlling TTY")
	}
}
