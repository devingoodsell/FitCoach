-- E3-S3: recorded session outcomes. Append-many per user. data holds the logged
-- session (exercises, reps, weight, rests, timed work, autoregulation events) as
-- a versioned JSON document. client_session_id is the idempotency key so an
-- offline client re-syncing the same session does not duplicate it (E12).
-- Cascades on account deletion.
CREATE TABLE workout_logs (
    id                BINARY(16)  NOT NULL PRIMARY KEY,
    user_id           BINARY(16)  NOT NULL,
    client_session_id VARCHAR(64) NOT NULL,
    schema_version    INT         NOT NULL,
    data              JSON        NOT NULL,
    performed_at      DATETIME(6) NOT NULL,
    created_at        DATETIME(6) NOT NULL,
    UNIQUE KEY uq_workout_idem (user_id, client_session_id),
    KEY idx_workout_user_time (user_id, performed_at),
    CONSTRAINT fk_workout_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
