CREATE TABLE IF NOT EXISTS pastes (
    id              VARCHAR(7)   PRIMARY KEY,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    views           INT          NOT NULL DEFAULT 0,
    expires_at      TIMESTAMPTZ,
    remaining_views INT,
    hashed_password TEXT,
    safety_token    VARCHAR(32)  NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS files (
    id              VARCHAR(7)   PRIMARY KEY,
    paste_id        VARCHAR(7)   NOT NULL REFERENCES pastes(id) ON DELETE CASCADE,
    name            TEXT,
    language        TEXT,
    content         TEXT         NOT NULL,
    character_count INT          NOT NULL,
    line_count      INT          NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_files_paste_id ON files(paste_id);
