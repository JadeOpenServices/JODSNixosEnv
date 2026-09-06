# Local engineering agent

`aiEnable=false` omits Ollama, model provisioning, OpenCode, research, and all
agent launchers. With AI enabled, the installer resolves exactly one base model.
`overrideAiSelection=false` ignores the configured model and uses the hardware
table in `internal/ai/profile/profile.go`. `overrideAiSelection=true` requires a
non-empty, valid `overrideModelWith` and uses it exactly.

Automatic profiles, strongest first:

| Profile | Requirements | Base model | Context |
|---|---|---|---|
| dedicated | 32 GiB RAM, 8 CPU threads, dedicated GPU, 12 GiB VRAM | `qwen3-coder:30b` | 32768 |
| integrated | 16 GiB RAM, 4 CPU threads | `qwen2.5-coder:14b` | 16384 |
| low-memory | fallback | `qwen2.5-coder:7b` | 8192 |

The resolved value is written once as `settings.aiModel`. Ollama binds only to
`127.0.0.1:11434`. `ollama-model-provision.service` waits at most 30 seconds for
Ollama, pulls the base model, then creates `gjallaros-caveman-ai`. A SHA-256 of
the base model, context, and system definition is stored only after successful
creation. An unchanged definition is not recreated.

`opencode-local` and `gjallar-ai` both start the isolated OpenCode runtime with
the derived model. Cloud providers, sharing, generic network tools, and generic
shell are disabled. The runtime sees the active Git workspace, temporary home,
localhost Ollama, and the research Unix socket.

## Policy and trust boundaries

`gjallar-agent-tool` calls the Go policy/execution layer. The model cannot alter
its classifications. Named capabilities cover Git inspection, repository
search, bounded tests/format checks, Nix evaluation/build/check/dry-build,
system inspection, service management, and deployment. Commands have finite
timeouts, a sanitized environment, exit reporting, and JSONL audit records.

Risk levels are: read-only (1), workspace mutation (2), bounded validation (3),
privileged/live change (4), and sensitive/destructive/external (5). Workspace
mode allows levels 1–3. Live deployment and service changes require an exact,
five-minute, one-use approval made interactively with `gjallar-ai-approve`.
Destructive Git, arbitrary network shell, external writes, and secret paths are
denied. Approval for one argv sequence never approves another.

`aiAgentMode` may be `workspace`, `owner-conservative`, or `owner-full-local`.
Owner mode changes policy decisions but never bypasses policy. The full-local
profile can auto-approve local service/deployment operations; arbitrary outbound
shell, external writes, credentials, and policy self-modification remain
separate protected boundaries. The active mode is printed on startup.

Repository files, web pages, model text, logs, and tool output are untrusted
data. They cannot grant capabilities. The read-only research broker accepts
credential-free HTTPS URLs only for explicit official allowlisted domains,
rejects private/link-local DNS answers and unsafe redirects, limits retrieval
to 2 MiB/15 seconds, and returns JSON with URL, source, update header, retrieval
time, and content. It has no cookies, JavaScript, forms, upload, or authenticated
browser state.

Run `gjallar-ai-diagnostics` for one bounded report covering the resolved model,
context/VRAM, mode, services, API, installed models, definition hash, and
OpenCode availability. Extend the system by adding a narrow capability and its
tests to `internal/ai/policy` and `internal/ai/agentexec`; never add a generic
unrestricted command or network runner.
