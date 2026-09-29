#!/usr/bin/env python3
"""Exercise source preparation using local Git repositories, never a host install."""

import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


BOOTSTRAP = Path(__file__).resolve().parents[1] / "installation/install.sh"


class BootstrapSourceTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="gjallar-bootstrap-")
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.env = dict(os.environ, GIT_CONFIG_GLOBAL="/dev/null",
                        GIT_CONFIG_NOSYSTEM="1", GIT_ALLOW_PROTOCOL="file")
        tools = self.root / "bin"
        tools.mkdir()
        nix = tools / "nix"
        nix.write_text('''#!/usr/bin/env bash
set -eu
test -f "$TEST_REPO/pkgs/monique/nix/nixos-module.nix"
touch "$TEST_REPO/nix-invoked"
for arg in "$@"; do
    case "$arg" in path:*)
        dir="${arg#path:}"; dir="${dir%%#*}"
        test "$dir" != "$TEST_REPO"
        test ! -e "$dir/.git"
        test -f "$dir/pkgs/monique/nix/nixos-module.nix"
        printf '%s\n' "$dir" >> "$TEST_REPO/nix-flake-dirs";;
    esac
done
case " $* " in *" --print-out-paths "*) printf '/unused-test-output\\n';; esac
''')
        nix.chmod(0o755)
        self.env["PATH"] = str(tools) + os.pathsep + self.env["PATH"]

    def git(self, repo, *args):
        return subprocess.run(["git", "-C", str(repo), "-c", "user.name=Test",
                               "-c", "user.email=test@example.invalid", *args],
                              env=self.env, check=True, capture_output=True)

    def install_script(self, repo):
        script = repo / "scripts/installation/install.sh"
        script.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(BOOTSTRAP, script)
        return script

    def run_bootstrap(self, repo):
        return subprocess.run(["bash", str(repo / "scripts/installation/install.sh")],
                              env=dict(self.env, TEST_REPO=str(repo)),
                              text=True, capture_output=True)

    def test_plain_clone_initializes_pinned_submodule_before_nix(self):
        module = self.root / "module"
        module.mkdir()
        self.git(module, "init", "-q")
        (module / "nix").mkdir()
        (module / "nix/nixos-module.nix").write_text("{}\n")
        self.git(module, "add", ".")
        self.git(module, "commit", "-qm", "module")
        origin = self.root / "origin"
        origin.mkdir()
        self.git(origin, "init", "-q")
        self.install_script(origin)
        self.git(origin, "submodule", "add", str(module), "pkgs/monique")
        self.git(origin, "add", ".")
        self.git(origin, "commit", "-qm", "repository")
        clone = self.root / "clone"
        self.git(self.root, "clone", str(origin), str(clone))
        self.assertFalse((clone / "pkgs/monique/nix/nixos-module.nix").exists())
        result = self.run_bootstrap(clone)
        self.assertEqual(result.returncode, 0, result.stdout + result.stderr)
        self.assertTrue((clone / "nix-invoked").exists())
        self.assertEqual(self.git(clone / "pkgs/monique", "rev-parse", "HEAD").stdout,
                         self.git(module, "rev-parse", "HEAD").stdout)
        # Nix saw only a staged copy without git history, removed afterwards.
        staged = set((clone / "nix-flake-dirs").read_text().split())
        self.assertTrue(staged)
        for directory in staged:
            self.assertFalse(Path(directory).exists(), directory)

    def test_incomplete_copy_fails_before_nix_or_host_changes(self):
        repo = self.root / "copy"
        self.install_script(repo)
        result = self.run_bootstrap(repo)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("copy the complete repository", result.stderr)
        self.assertFalse((repo / "nix-invoked").exists())


if __name__ == "__main__":
    unittest.main()
