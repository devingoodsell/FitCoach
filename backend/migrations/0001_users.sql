-- E1-S1: accounts. One row per user; email_norm (lowercased) carries the unique
-- constraint so case variants can't create duplicate accounts. id is a UUIDv7
-- stored as BINARY(16); every user-owned table cascades from here so account
-- deletion (E1-S5) is correct by construction.
CREATE TABLE users (
    id             BINARY(16)   NOT NULL PRIMARY KEY,
    email          VARCHAR(320) NOT NULL,
    email_norm     VARCHAR(320) NOT NULL,
    password_hash  VARCHAR(255) NOT NULL,
    email_verified TINYINT(1)   NOT NULL DEFAULT 0,
    created_at     DATETIME(6)  NOT NULL,
    updated_at     DATETIME(6)  NOT NULL,
    UNIQUE KEY uq_users_email_norm (email_norm)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
