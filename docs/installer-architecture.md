# GjallarOS installer architecture

The installer is a typed Go application. `scripts/installation/install.sh`
only builds and launches it with `nix build`; it never opens a temporary Nix
environment. Persistent prerequisite installation remains owned by Go.

## Layout

```
cmd/gjallar-installer/       interactive installer entrypoint
cmd/gjallarctl/              non-interactive diagnostics and maintenance
internal/installer/
  config/                    typed preset and generated-settings model
  discovery/                 read-only hardware and repository discovery
  nixrender/                 safe Nix-literal rendering
  bootstrap/, deploy/,
  diskcrypto/, ...           explicit, auditable privileged operations
  prompt/                    terminal/GTK prompts behind one interface
```

## Module contract

Every module has a narrow input/output type, unit tests, and one owner. No
module executes a shell. Privileged action packages must validate their
target, use fixed argument vectors and atomic
writes, preserve an existing backup where appropriate, and return an error
rather than continuing after a failed command.

## Security rules

- Parse JSON with `encoding/json`; never use `sed`, `grep`, `eval`, or shell
  expansion to interpret configuration.
- Build external commands with fixed argument arrays; never invoke `sh -c`.
- Treat all preset strings as untrusted. Render Nix values through
  `nixrender.String`; it escapes `${...}` as well as quotes and backslashes.
- Keep secrets out of generated settings and logs. Password hashes are
  root-owned files; plaintext remains in memory only for the shortest
  possible time.
- Print the exact plan before privileged or destructive operations and require
  explicit confirmation or `--apply`.

## Adding a module

Copy this shape:

```go
package example

type Input struct { /* validated values */ }
type Result struct { /* no hidden global state */ }

func Run(ctx context.Context, input Input) (Result, error) {
    // No shell. No filesystem writes outside action/.
}
```

Add table tests for valid input, invalid input, and the empty/hardware-missing
case. Wire the module through the Go entrypoint before extending the legacy
compatibility adapters.
