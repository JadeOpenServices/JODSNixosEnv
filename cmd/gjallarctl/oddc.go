package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// The Nix package sets both through -ldflags; the defaults suit a
// development checkout.
var (
	oddcctlPath = "oddcctl"
	oddcRoot    = "oddc"
)

// oddcctlCommands pass through to oddcctl, which owns them. gjallarctl
// points them at the ODDC deployment of this system: only the selected
// model's closure, not the catalog. resolve and explain default to that
// model and its host overlay.
var oddcctlCommands = []string{"validate", "list", "resolve", "explain"}

func hasFlag(args []string, name string) bool {
	return slices.ContainsFunc(args, func(arg string) bool {
		return arg == name || strings.HasPrefix(arg, name+"=")
	})
}

func deployedModel(root string) string {
	data, err := os.ReadFile(filepath.Join(root, "resolved.json"))
	if err != nil {
		return ""
	}
	var resolved struct {
		Model struct {
			ID string `json:"id"`
		} `json:"model"`
	}
	if json.Unmarshal(data, &resolved) != nil {
		return ""
	}
	return resolved.Model.ID
}

func runODDCctl(args []string, stdout, stderr io.Writer) int {
	argv := slices.Clone(args)
	root := oddcRoot
	if !hasFlag(argv, "--root") {
		argv = append(argv, "--root", oddcRoot)
	} else {
		root = ""
	}
	if root != "" && (args[0] == "resolve" || args[0] == "explain") {
		if !hasFlag(argv, "--device") {
			if model := deployedModel(root); model != "" {
				argv = append(argv, "--device", model)
			}
		}
		overlay := filepath.Join(root, "host-overlay.json")
		if _, err := os.Stat(overlay); err == nil && !hasFlag(argv, "--host") {
			argv = append(argv, "--host", overlay)
		}
	}

	cmd := exec.Command(oddcctlPath, argv...)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return exit.ExitCode()
		}
		fmt.Fprintf(stderr, "ERROR: run oddcctl: %v\n", err)
		return 1
	}
	return 0
}

func isODDCctlCommand(name string) bool {
	return slices.Contains(oddcctlCommands, name)
}
