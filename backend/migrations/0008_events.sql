-- E15-S3: generation & safety event log (audit/observability). payload is stored
-- already redacted (no secrets, no prompt contents). On account deletion the
-- user link is nulled rather than cascaded so the de-identified audit trail
-- survives; since payload carries no PII this still honors "deletion deletes".
CREATE TABLE events (
    id         BINARY(16)  NOT NULL PRIMARY KEY,
    user_id    BINARY(16)  NULL,
    type       VARCHAR(50) NOT NULL,
    payload    JSON        NOT NULL,
    created_at DATETIME(6) NOT NULL,
    KEY idx_events_user (user_id),
    KEY idx_events_type_time (type, created_at),
    CONSTRAINT fk_events_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
