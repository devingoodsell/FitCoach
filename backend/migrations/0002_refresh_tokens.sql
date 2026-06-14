-- E1-S2/S6: refresh-token sessions. One row per active session (device). The
-- opaque token is never stored; only its SHA-256 hash. Sessions are revocable
-- (logout, password reset) and rotated on refresh. Cascades on account deletion.
CREATE TABLE refresh_tokens (
    id           BINARY(16)   NOT NULL PRIMARY KEY,
    user_id      BINARY(16)   NOT NULL,
    token_hash   CHAR(64)     NOT NULL,
    device_label VARCHAR(100) NULL,
    expires_at   DATETIME(6)  NOT NULL,
    revoked_at   DATETIME(6)  NULL,
    created_at   DATETIME(6)  NOT NULL,
    UNIQUE KEY uq_refresh_token_hash (token_hash),
    KEY idx_refresh_user (user_id),
    CONSTRAINT fk_refresh_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
