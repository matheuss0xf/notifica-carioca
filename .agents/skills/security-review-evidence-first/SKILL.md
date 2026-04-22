---
name: security-review-evidence-first
description: Project-local security review workflow that reports only traced, actionable findings.
source:
  marketplace: getsentry-sentry-sentry-security
---

# Security Review Evidence First

Use this skill when:

- reviewing API endpoints for IDOR or access-control bugs
- auditing auth/authz changes
- checking token handling, ownership, or scope boundaries
- doing a targeted security review of handlers, middleware, or repository access

## Review Standard

Report only findings with real code evidence.

- **HIGH**: traced end-to-end and confirmed missing enforcement
- **MEDIUM**: strong concern, but one enforcement layer still needs confirmation
- **LOW / theoretical**: do not report

Do not fill the report with generic OWASP advice if the code does not support it.

## What To Trace

For every request-derived identifier or token:

1. Where does it enter? URL, query, header, or JSON body.
2. Where is it used? Handler, middleware, service, repository, or DB query.
3. Which enforcement exists between entry and use?
4. Is the object/action scoped to the authenticated user or trusted context?

## Highest-Priority Checks For This Project

### 1. IDOR / Ownership Drift

- Any ID from request parameters must stay scoped to the authenticated `cpf_hash`.
- Pagination anchors and lookup IDs must not reveal or traverse another user's data.

### 2. Missing Authorization

- Authentication is not enough.
- Confirm ownership or allowed scope on each resource operation.

### 3. Token Handling

- Prefer `Authorization` header over query-string token usage.
- If query fallback exists, surrounding checks must be stricter.
- Do not log raw tokens or credentials.

### 4. Browser-Facing Security

- WebSocket origin rules must be explicit.
- Public endpoints should not ship permissive defaults without a config guard.

### 5. Input and Error Surfaces

- Validate IDs, cursors, and limits early.
- Avoid error responses that leak internal state or foreign-resource existence.

## Enforcement Chain

Trace the full path before reporting:

1. inbound middleware
2. handler
3. application service
4. repository / DB query
5. outbound response or side effect

A check at any layer counts as enforcement. If you cannot confirm the full chain is missing, keep the finding at MEDIUM or drop it.

## Expected Output

For each finding:

- category
- exact file/location
- confidence
- short traced flow
- concrete impact
- concrete fix direction

If no traced findings exist, say so plainly.
