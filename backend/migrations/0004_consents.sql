-- E1-S4: health-data consent. Append-only log of accepted consent/disclaimer
-- versions; the latest row per (user, type) is the current state. Records the
-- exact version accepted so disclaimer changes are auditable. Cascades on
-- account deletion.
CREATE TABLE consents (
    id          BINARY(16)  NOT NULL PRIMARY KEY,
    user_id     BINARY(16)  NOT NULL,
    type        VARCHAR(50) NOT NULL,
    version     VARCHAR(20) NOT NULL,
    accepted_at DATETIME(6) NOT NULL,
    KEY idx_consent_user (user_id),
    CONSTRAINT fk_consent_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
