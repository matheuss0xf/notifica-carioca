---
name: api-security-best-practices
description: Project-local API security guidance for Notifica Carioca.
source:
  marketplace: davila7-claude-code-templates-api-security-best-practices
---

# API Security Best Practices

Use this skill when:

- designing or changing REST endpoints
- changing JWT auth or ownership checks
- reviewing WebSocket authentication or browser-facing behavior
- adding webhook handling, request validation, or abuse controls
- performing API security reviews

## Core Rules

1. Authenticate every protected endpoint.
2. Enforce authorization and ownership on every resource access.
3. Validate all inbound data and reject malformed input early.
4. Keep error responses sanitized; do not leak internal details.
5. Prefer headers over query params for credentials when possible.
6. Treat browser-facing origin/CORS rules as explicit configuration, not permissive defaults.
7. Add abuse controls for public endpoints when exposure grows.

## This Project

Key API surfaces:

- `POST /api/v1/webhooks/status-change`
- `GET /api/v1/notifications`
- `PATCH /api/v1/notifications/:id/read`
- `GET /api/v1/notifications/unread-count`
- `GET /ws`

## Review Checklist

- Is the caller authenticated where required?
- Is resource ownership enforced using `cpf_hash`?
- Are IDs, limits, cursors, and payloads validated?
- Are token parsing and origin checks explicit and conservative?
- Do error responses avoid exposing internals?
- Would this endpoint benefit from rate limiting or additional monitoring?

## Project Guidance

### Authentication and Authorization

- JWT validation must reject invalid method, invalid claims, and missing identity.
- Notification reads and pagination must stay scoped to the authenticated citizen.
- Do not assume authentication alone is enough; enforce ownership on each resource action.

### Validation

- Validate IDs, cursors, and numeric limits explicitly.
- For JSON payloads, keep required fields explicit and reject malformed bodies early.
- Keep SQL access parameterized and repository-driven.

### WebSocket

- Do not allow all origins by default.
- If query-string token fallback is supported, keep surrounding checks stricter.
- Be careful not to expose a browser-usable cross-origin path without explicit approval.

### Errors and Logging

- Return stable, generic API errors to clients.
- Log enough for operators, but never log secrets or raw credentials.
- Avoid responses that reveal whether a protected foreign resource exists.

### Abuse Controls

- Public endpoints should consider per-IP or per-identity rate limits.
- Authentication and webhook endpoints are higher-priority candidates for throttling.

## Expected Output For API Changes

When changing API behavior in this repo:

1. Call out auth/authz impact.
2. Note validation changes and failure modes.
3. Mention origin/CORS/rate-limit implications when relevant.
4. Add regression tests for ownership and invalid-input paths when practical.
