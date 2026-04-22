---
name: hexagonal-architecture
description: Project-local Ports & Adapters guidance for Notifica Carioca.
source:
  marketplace: affaan-m-everything-claude-code-hexagonal-architecture
---

# Hexagonal Architecture

Use this skill when:

- adding a new feature slice
- refactoring handlers, services, repositories, or adapters
- introducing a new external dependency
- moving logic across domain, application, and adapter boundaries

## This Project's Shape

- `internal/domain`
  Business entities and domain errors. No framework or infrastructure imports.
- `internal/application`
  Use-case orchestration and ports.
- `internal/adapters/in`
  Inbound transport adapters such as HTTP handlers and middleware.
- `internal/adapters/out`
  Outbound infrastructure adapters such as PostgreSQL, Redis, and WebSocket.
- `cmd/api/main.go`
  Composition root and wiring.

## Core Rules

1. Dependency direction stays inward.
2. Domain must not know Gin, Redis, PostgreSQL, or WebSocket details.
3. Application orchestrates behavior through ports, not through concrete infra types.
4. Adapters translate protocol and storage concerns at the edges.
5. Wiring belongs in one explicit composition root.

## What Good Looks Like

- Handler receives HTTP input and converts it to plain use-case input.
- Application service coordinates domain logic and port calls.
- Repository, cache, publisher, and transport implementations live in outbound adapters.
- Port interfaces stay small and consumer-owned.

## Refactor Checklist

- Did any framework type leak into domain or application?
- Did a use case start depending directly on an adapter implementation?
- Did mapping logic stay in adapters instead of spreading into domain rules?
- Is the composition root still the only place where concrete dependencies are assembled?
- Can the use case still be tested with small stubs/fakes?

## Project Guidance

### Inbound side

- Keep Gin-specific parsing, headers, and response formatting in handlers/middleware.
- Do not let handlers own business decisions that belong in the application layer.

### Outbound side

- Keep SQL, Redis, and Pub/Sub details inside outbound adapters.
- Ports should describe capabilities such as repository access, event publishing, caching, and hashing.

### Use cases

- Keep orchestration in `internal/application`.
- Prefer explicit inputs/outputs and contextual error wrapping.

## When Adding New Capabilities

If a feature needs a new side effect:

1. define or extend an outbound port
2. implement it in an adapter
3. inject it through the composition root
4. test the use case with a fake port

## Anti-Patterns

- Business logic in handlers
- Repositories called directly from middleware when that bypasses use cases
- Domain objects importing framework or storage packages
- Hidden globals or scattered wiring
