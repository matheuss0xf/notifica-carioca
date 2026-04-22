---
name: golang-testing
description: Project-local Go testing guidance for Notifica Carioca.
source:
  marketplace: affaan-m-everything-claude-code-golang-testing
---

# Go Testing

Use this skill when:

- adding regression coverage
- changing Go business logic
- fixing bugs in handlers, repositories, or concurrency code
- introducing new public behavior

## Test Style

1. Prefer table-driven tests for branching behavior.
2. Use `t.Run(...)` subtests with descriptive names.
3. Test behavior, not implementation details.
4. Use small handwritten mocks/stubs for ports and adapters.
5. Keep tests deterministic. Do not use `time.Sleep` for synchronization.
6. Add focused regression tests for every bug fix.

## Project Priorities

Prioritize tests for:

- webhook idempotency behavior
- pagination and ownership boundaries
- unread count cache invalidation flows
- WebSocket hub concurrency and delivery behavior
- config parsing and security-sensitive defaults

## Suggested Patterns

### Table-driven behavior tests

Use for:

- input validation
- origin allowlists
- cursor handling
- limit parsing

### Small stub implementations

Use local stubs for:

- `NotificationRepository`
- `UnreadCache`
- `IdempotencyStore`
- `EventPublisher`
- auth/token helpers

### Concurrency-sensitive code

- Avoid flaky tests.
- Favor snapshot-based assertions over timing-sensitive coordination.
- Test externally visible effects, such as message delivery and safe unregister behavior.

## Minimum Validation

When Go tooling is available, run:

```bash
go test ./...
go test -race ./...
```

If coverage is being improved as part of the task, also run:

```bash
go test -cover ./...
```
