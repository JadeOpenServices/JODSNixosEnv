# Local AI

`aiEnable` controls the complete AI feature set. When false, Ollama, wrappers,
profiling, and AI services are omitted.

When enabled, Ollama binds to `127.0.0.1`, limits concurrency and loaded
models, and selects a model from detected RAM/VRAM. `gjallar-ai` runs it.
Retrieval is explicit HTTPS only and follows
`system/apps/ai/retrieval-policy.json`. Agent instructions favor concise,
meaning-preserving output while keeping commands and safety details exact.
