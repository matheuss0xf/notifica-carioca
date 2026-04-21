CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE notifications (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chamado_id      VARCHAR(50)  NOT NULL,
    cpf_hash        VARCHAR(64)  NOT NULL,
    tipo            VARCHAR(50)  NOT NULL DEFAULT 'status_change',
    status_anterior VARCHAR(30),
    status_novo     VARCHAR(30)  NOT NULL,
    titulo          TEXT         NOT NULL,
    descricao       TEXT,
    read_at         TIMESTAMPTZ,
    event_timestamp TIMESTAMPTZ  NOT NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    -- Idempotency: same chamado + same status + same timestamp = duplicate
    CONSTRAINT uq_notification_event
        UNIQUE(chamado_id, status_novo, event_timestamp)
);

-- Listing notifications by citizen (cursor pagination ordered by created_at DESC)
CREATE INDEX idx_notifications_cpf_created
    ON notifications(cpf_hash, created_at DESC);

-- Counting unread notifications efficiently (partial index)
CREATE INDEX idx_notifications_cpf_unread
    ON notifications(cpf_hash)
    WHERE read_at IS NULL;
