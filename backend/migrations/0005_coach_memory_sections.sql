-- E3-S1: Coach Memory, singleton sections. One row per (user, section); each
-- section is a versioned JSON document so the schema can evolve without data
-- loss (in-code upgrader migrates older schema_version rows forward on read).
-- Sections: profile, goals, schedule, preferences, locations, injuries, diet.
-- Cascades on account deletion (E1-S5).
CREATE TABLE coach_memory_sections (
    user_id        BINARY(16)  NOT NULL,
    section        VARCHAR(32) NOT NULL,
    schema_version INT         NOT NULL,
    data           JSON        NOT NULL,
    created_at     DATETIME(6) NOT NULL,
    updated_at     DATETIME(6) NOT NULL,
    PRIMARY KEY (user_id, section),
    CONSTRAINT fk_memory_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
