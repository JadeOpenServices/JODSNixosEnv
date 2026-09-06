package agentexec

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestWorkspaceFlakeTarget(t *testing.T) {
	workspace := filepath.Clean("/workspace/repo")
	base := "path:" + workspace

	good := map[string]string{
		"":                                base,
		".":                               base,
		".#packages.x86_64-linux.default": base + "#packages.x86_64-linux.default",
		"#nixosConfigurations.gjallarOS.config.system.build.toplevel": base + "#nixosConfigurations.gjallarOS.config.system.build.toplevel",
		base:          base,
		base + "#foo": base + "#foo",
	}

	for input, want := range good {
		got, err := workspaceFlakeTarget(workspace, input)
		if err != nil {
			t.Fatalf("workspace target %q rejected: %v", input, err)
		}
		if got != want {
			t.Fatalf("workspace target %q = %q, want %q", input, got, want)
		}
	}

	bad := []string{
		"nixpkgs#hello",
		"github:NixOS/nixpkgs#hello",
		"git+https://example.invalid/repo",
		"https://example.invalid/source.tar.gz",
		"http://example.invalid/source.tar.gz",
		"gitlab:example/repo",
		"sourcehut:example/repo",
		"path:/tmp/evil#default",
		"/tmp/evil#default",
		"../other#default",
	}

	for _, input := range bad {
		if _, err := workspaceFlakeTarget(workspace, input); err == nil {
			t.Fatalf("external target %q unexpectedly accepted", input)
		}
	}
}

func TestBoundedNixFlagsRejectSourceChangingOptions(t *testing.T) {
	bad := []string{
		"--override-input",
		"--inputs-from",
		"--reference-lock-file",
		"--output-lock-file",
		"--update-input",
		"--recreate-lock-file",
		"--expr",
		"--file",
		"-f",
		"--include",
		"-I",
		"--option",
		"--impure",
		"--refresh",
		"--offline",
	}

	for _, arg := range bad {
		if _, err := boundedNixFlags([]string{arg}); err == nil {
			t.Fatalf("unsafe Nix option %q unexpectedly accepted", arg)
		}
	}
}

func TestNixBuildIsWorkspaceBound(t *testing.T) {
	workspace := "/workspace/repo"

	name, args, err := command(
		workspace,
		"nix-build",
		[]string{".#packages.x86_64-linux.default", "--show-trace"},
	)
	if err != nil {
		t.Fatal(err)
	}

	if name != "nix" {
		t.Fatalf("command = %q, want nix", name)
	}

	joined := strings.Join(args, " ")

	for _, required := range []string{
		"build",
		"--no-write-lock-file",
		"--no-update-lock-file",
		"--no-use-registries",
		"path:/workspace/repo#packages.x86_64-linux.default",
	} {
		if !strings.Contains(joined, required) {
			t.Fatalf("generated command missing %q: %s", required, joined)
		}
	}
}

func TestNixToolsRejectDirectExternalFetches(t *testing.T) {
	workspace := "/workspace/repo"

	cases := []struct {
		tool string
		args []string
	}{
		{"nix-eval", []string{"github:NixOS/nixpkgs#legacyPackages.x86_64-linux.hello"}},
		{"nix-build", []string{"nixpkgs#hello"}},
		{"nix-build", []string{"https://example.invalid/source.tar.gz"}},
		{"nix-build", []string{".#default", "--override-input"}},
		{"nix-build", []string{".#default", "--impure"}},
		{"nix-check", []string{"--override-input"}},
		{"nixos-dry-build", []string{"github:evil/example#host"}},
		{"nixos-deploy", []string{"switch", "git+https://example.invalid/repo#host"}},
	}

	for _, tc := range cases {
		if _, _, err := command(workspace, tc.tool, tc.args); err == nil {
			t.Fatalf(
				"%s unexpectedly accepted external/unsafe args %#v",
				tc.tool,
				tc.args,
			)
		}
	}
}

func TestRebuildToolsAcceptWorkspaceTarget(t *testing.T) {
	workspace := "/workspace/repo"

	_, args, err := command(
		workspace,
		"nixos-dry-build",
		[]string{".#gjallarOS", "--show-trace"},
	)
	if err != nil {
		t.Fatalf("nixos-dry-build rejected workspace target: %v", err)
	}

	if !strings.Contains(
		strings.Join(args, " "),
		"path:/workspace/repo#gjallarOS",
	) {
		t.Fatalf("dry-build target escaped workspace: %v", args)
	}

	_, args, err = command(
		workspace,
		"nixos-deploy",
		[]string{"switch", ".#gjallarOS", "--show-trace"},
	)
	if err != nil {
		t.Fatalf("nixos-deploy rejected workspace target: %v", err)
	}

	if !strings.Contains(
		strings.Join(args, " "),
		"path:/workspace/repo#gjallarOS",
	) {
		t.Fatalf("deploy target not workspace-bound: %v", args)
	}
}
