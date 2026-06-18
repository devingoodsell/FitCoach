-- E14-S2: allow withdrawing a previously granted consent without losing the audit
-- trail. revoked_at NULL means in force; a timestamp means withdrawn (health-data
-- ingestion then falls back to manual mode). The index speeds the "is there an
-- active consent of this type?" gate (consent.HasConsent). Single statement: the
-- migration runner executes each file as one Exec.
ALTER TABLE consents
    ADD COLUMN revoked_at DATETIME(6) NULL AFTER accepted_at,
    ADD INDEX idx_consent_active (user_id, type, revoked_at);
