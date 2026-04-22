---
name: database-migrations
description: Project-local migration guidance for Notifica Carioca using golang-migrate.
source:
  marketplace: affaan-m-everything-claude-code-database-migrations
---

# Database Migrations

Use this skill when:

- creating or altering tables in PostgreSQL
- adding or removing columns or indexes
- changing uniqueness, constraints, or ownership semantics
- planning backfills or rollout-safe schema transitions
- reviewing SQL migration files under `migrations/`

## Core Rules

1. Every production schema change must be a migration file.
2. Treat deployed migrations as immutable.
3. Prefer forward fixes over editing old migrations.
4. Keep schema changes and large data backfills separate.
5. Favor additive, rollout-safe changes before destructive cleanup.

## This Project

- Migration tool: `golang-migrate`
- Directory: `migrations/`
- Naming pattern: sequential `NNN_description.up.sql` and `.down.sql`

## Review Checklist

- Does the change have both `up` and `down` files?
- Is the migration safe on an existing table with data?
- Does it avoid long blocking operations where possible?
- Does it preserve idempotency and ownership guarantees used by the app?
- If data must move, can that happen in a separate migration or operational step?
- Is the rollback story documented, even if the down migration is limited?

## Safe Defaults

### Adding columns

- Prefer nullable columns first.
- If using `NOT NULL`, include a safe default when the table may already contain rows.

### Indexes

- For existing PostgreSQL tables, prefer `CREATE INDEX CONCURRENTLY` when operationally appropriate.
- Remember that concurrent index creation cannot run inside a transaction block.

### Renames and removals

- Prefer expand-contract:
  1. add the new shape
  2. deploy code that can work with both
  3. backfill if needed
  4. remove old shape later

### Data migrations

- Avoid mixing large DML backfills into the same migration as DDL.
- For large updates, plan batched execution instead of one large transaction.

## Anti-Patterns

- Editing an old migration that may already have run elsewhere
- Adding `NOT NULL` columns without a safe default on live tables
- Dropping columns before application code has stopped using them
- Combining schema change and large backfill into one risky migration

## Commands

```bash
migrate create -ext sql -dir migrations -seq change_name
migrate -path migrations -database "$DATABASE_URL" up
migrate -path migrations -database "$DATABASE_URL" down 1
```

## Expected Output For Migration Work

When making a schema change in this repo:

1. Add the new migration pair under `migrations/`.
2. Explain rollout and rollback impact briefly.
3. Call out any zero-downtime caveats explicitly.
