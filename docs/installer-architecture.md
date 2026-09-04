# GjallarOS installer architecture

The installer is being migrated from sourced Bash functions to a typed Go
application. During the migration, Bash is an entrypoint/compatibility layer
only; policy, parsing, rendering, discovery, and validation live in Go.

## Layout

```
cmd/gjallar-installer/       interactive installer entrypoint
cmd/gjallarctl/              non-interactive diagnostics and maintenance
internal/installer/
  config/                    typed preset and generated-settings model
  discover/                  read-only hardware and repository discovery
  nixrender/                 safe Nix-literal rendering
  action/                    explicit, auditable privileged operations
  ui/                        terminal/GTK prompts behind one interface
```

## Module contract

Every module has a narrow input/output type, unit tests, and one owner. A
module may not execute a shell, use `sudo`, or mutate files unless it is in
`internal/installer/action`. Actions must validate their target, use atomic
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
- Keep destructive operations explicit: `--apply` is required after a
  read-only plan/dry-run has succeeded.

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
case. Wire the module through the Go command, then retire its Bash adapter only
after behavior-equivalence tests pass.
