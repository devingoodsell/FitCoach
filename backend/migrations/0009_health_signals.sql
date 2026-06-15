-- E4-S1/S2: raw recovery signals ingested from Health Connect (account-scoped,
-- timestamped). We store raw samples and compute our OWN readiness from them
-- (E4-S2) rather than persisting a vendor score. `day` is the local date the
-- sample pertains to (overnight metrics attribute to the wake day). Uploads are
-- gated by health-data consent (E1-S4). Cascades on account deletion.
CREATE TABLE health_signals (
    id         BINARY(16)  NOT NULL PRIMARY KEY,
    user_id    BINARY(16)  NOT NULL,
    kind       VARCHAR(32) NOT NULL,  -- hrv_ms | rhr_bpm | sleep_minutes
    value      DOUBLE      NOT NULL,
    day        DATE        NOT NULL,
    created_at DATETIME(6) NOT NULL,
    UNIQUE KEY uq_signal_user_kind_day (user_id, kind, day),
    KEY idx_signal_user_day (user_id, day),
    CONSTRAINT fk_signal_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
