CREATE TABLE IF NOT EXISTS pastes (
    id              TEXT   PRIMARY KEY,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    web             BOOLEAN NOT NULL,
    views           INT          NOT NULL DEFAULT 0,
    expires_at      TIMESTAMPTZ,
    deleted_at      TIMESTAMPTZ,
    remaining_views INT,
    hashed_password TEXT,
    safety_token    TEXT  NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS files (
    id              TEXT   PRIMARY KEY,
    paste_id        TEXT   NOT NULL REFERENCES pastes(id) ON DELETE CASCADE,
    name            TEXT,
    language        TEXT,
    content         TEXT         NOT NULL,
    deleted_at      TIMESTAMPTZ,
    character_count INT          NOT NULL,
    line_count      INT          NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_files_paste_id ON files(paste_id);
CREATE INDEX IF NOT EXISTS idx_pastes_deleted_at ON pastes(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_files_deleted_at ON files(deleted_at) WHERE deleted_at IS NOT NULL;
