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

CREATE TABLE IF NOT EXISTS api_request_logs (
    id             BIGSERIAL PRIMARY KEY,
    requested_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- Intentionally not a foreign key: request history remains after a paste is deleted.
    paste_id       TEXT,
    client_ip      INET        NOT NULL,
    method         TEXT        NOT NULL,
    route          TEXT        NOT NULL,
    status_code    SMALLINT    NOT NULL,
    latency_us     BIGINT      NOT NULL,
    response_bytes BIGINT      NOT NULL,
    user_agent     TEXT
);

CREATE INDEX IF NOT EXISTS idx_files_paste_id ON files(paste_id);
CREATE INDEX IF NOT EXISTS idx_pastes_deleted_at ON pastes(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_files_deleted_at ON files(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_api_request_logs_requested_at ON api_request_logs(requested_at);
CREATE INDEX IF NOT EXISTS idx_api_request_logs_paste_id_requested_at
    ON api_request_logs(paste_id, requested_at DESC) WHERE paste_id IS NOT NULL;
