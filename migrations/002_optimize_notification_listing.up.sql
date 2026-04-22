DROP INDEX IF EXISTS idx_notifications_cpf_created;

CREATE INDEX idx_notifications_cpf_created_id
    ON notifications(cpf_hash, created_at DESC, id DESC);
