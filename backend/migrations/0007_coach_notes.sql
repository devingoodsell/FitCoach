-- E3-S1: coach notes. Append-many free-form structured notes the coach accrues
-- about a user (versioned JSON). Cascades on account deletion.
CREATE TABLE coach_notes (
    id             BINARY(16)  NOT NULL PRIMARY KEY,
    user_id        BINARY(16)  NOT NULL,
    schema_version INT         NOT NULL,
    data           JSON        NOT NULL,
    created_at     DATETIME(6) NOT NULL,
    KEY idx_notes_user (user_id),
    CONSTRAINT fk_notes_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
