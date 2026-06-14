-- E1-S3: password reset. Single-use, time-limited tokens stored as SHA-256
-- hashes (never plaintext). used_at enforces single use; expires_at enforces the
-- time limit. Cascades on account deletion.
CREATE TABLE password_reset_tokens (
    id         BINARY(16)  NOT NULL PRIMARY KEY,
    user_id    BINARY(16)  NOT NULL,
    token_hash CHAR(64)    NOT NULL,
    expires_at DATETIME(6) NOT NULL,
    used_at    DATETIME(6) NULL,
    created_at DATETIME(6) NOT NULL,
    UNIQUE KEY uq_reset_token_hash (token_hash),
    KEY idx_reset_user (user_id),
    CONSTRAINT fk_reset_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
