DROP INDEX IF EXISTS idx_notifications_cpf_created_id;

CREATE INDEX idx_notifications_cpf_created
    ON notifications(cpf_hash, created_at DESC);
