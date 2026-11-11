-- Core identity: accounts, sessions, settings and app links.
-- Owner: backend/internal/core/auth and backend/internal/core/user.

CREATE TABLE users (
    id                  bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    telegram_id_lookup  text        NOT NULL UNIQUE,
    telegram_id_sealed  bytea       NOT NULL,
    key_id              text        NOT NULL,
    public_name         text        NOT NULL,
    username            text,
    status              text        NOT NULL DEFAULT 'active'
                        CHECK (status IN ('active', 'blocked', 'deleted')),
    created_at          timestamptz NOT NULL DEFAULT now(),
    updated_at          timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE user_settings (
    user_id         bigint PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    lang            text        NOT NULL DEFAULT 'ru' CHECK (lang IN ('ru', 'kk', 'en')),
    theme           text        NOT NULL DEFAULT 'auto' CHECK (theme IN ('auto', 'light', 'dark')),
    timezone        text        NOT NULL DEFAULT 'Asia/Qyzylorda',
    pinned_apps     text[]      NOT NULL DEFAULT '{}',
    updated_at      timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    id          bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    token_hash  text        NOT NULL UNIQUE,
    user_id     bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at  timestamptz NOT NULL,
    created_at  timestamptz NOT NULL DEFAULT now(),
    revoked_at  timestamptz
);

CREATE INDEX sessions_user_idx ON sessions (user_id) WHERE revoked_at IS NULL;

-- Confirmed link between a SruNal account and an external app account.
CREATE TABLE app_links (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id       bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    app_id        text        NOT NULL CHECK (app_id IN ('doc', 'day', 'pulse', 'notes', 'split')),
    external_id   text        NOT NULL,
    linked_at     timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, app_id)
);

-- Idempotent cross-app action: the same key returns the stored result.
CREATE TABLE integration_requests (
    id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    idempotency_key text        NOT NULL UNIQUE,
    user_id         bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    source_app      text        NOT NULL,
    target_app      text        NOT NULL,
    content_hash    text        NOT NULL,
    result          jsonb,
    created_at      timestamptz NOT NULL DEFAULT now()
);
