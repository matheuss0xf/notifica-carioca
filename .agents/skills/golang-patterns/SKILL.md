---
name: golang-patterns
description: Project-local Go coding and review guidance for Notifica Carioca.
source:
  marketplace: affaan-m-everything-claude-code-golang-patterns
---

# Go Patterns

Use this skill when:

- writing or refactoring Go code
- reviewing Go changes
- changing concurrency, persistence, or API behavior

## Core Rules

1. Prefer simple and explicit code over clever code.
2. Wrap errors with context using `%w`.
3. Accept interfaces where useful, but return concrete types.
4. Keep types usable with zero values when practical.
5. Pass `context.Context` explicitly as the first parameter.
6. Avoid package-level mutable state.
7. Keep interfaces small and consumer-driven.

## Review Checklist

- Is the control flow straightforward?
- Are error paths explicit and contextual?
- Are domain and adapter boundaries still clear?
- Is ownership enforced on notification reads and pagination?
- Does Redis fast-path logic match PostgreSQL source-of-truth semantics?
- Could a goroutine, channel, or map access race under load?
- Does the change preserve privacy guarantees around CPF handling?

## Project Guidance

### Concurrency

- Do not iterate shared maps while another goroutine may mutate them.
- Prefer taking a snapshot under lock before broadcasting to WebSocket clients.
- Be careful with client disconnect paths, channel closing, and duplicate unregisters.

### Persistence and Idempotency

- Keep deduplication keys aligned with database uniqueness precision.
- Cursor pagination must stay scoped to the authenticated `cpf_hash`.
- Avoid behavior where Redis can drop valid events before PostgreSQL sees them.

### HTTP and WebSocket

- Do not broaden auth or origin rules without explicit configuration.
- Treat query-string tokens as higher risk than headers; keep surrounding checks tight.
- Keep handler error responses simple and stable.

## Tooling

When the environment supports it, run:

```bash
gofmt -w .
go test ./...
go test -race ./...
go test -cover ./...
```
