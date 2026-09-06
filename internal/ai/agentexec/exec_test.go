package agentexec

import "testing"

func TestNarrowCommands(t *testing.T) {
	tests := []struct {
		tool string
		args []string
		ok   bool
	}{
		{"git-inspect", []string{"status", "--short"}, true},
		{"git-inspect", []string{"push"}, false},
		{"system-inspect", []string{"status", "ollama.service"}, true},
		{"system-inspect", []string{"restart", "ollama.service"}, false},
		{"service-manage", []string{"restart", "ollama.service"}, true},
		{"service-manage", []string{"enable", "ollama.service"}, false},
		{"nixos-deploy", []string{"switch", ".#host"}, true},
		{"nixos-deploy", []string{"switch"}, false},
		{"local-command", []string{"journalctl", "-b"}, true},
		{"local-command", []string{"/bin/sh", "-c", "anything"}, false},
	}
	for _, tt := range tests {
		_, _, err := command("/work", tt.tool, tt.args)
		if (err == nil) != tt.ok {
			t.Fatalf("%s %v: err=%v", tt.tool, tt.args, err)
		}
	}
}
