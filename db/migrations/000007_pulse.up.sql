-- SruPulse: availability statuses, their audience and responses.
-- Owner: backend/internal/apps/pulse.

CREATE TABLE pulse_statuses (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    author_id    bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    text_sealed  bytea       NOT NULL,
    key_id       text        NOT NULL,
    expires_at   timestamptz NOT NULL,
    closed_at    timestamptz,
    created_at   timestamptz NOT NULL DEFAULT now(),
    CHECK (expires_at > created_at)
);

CREATE INDEX pulse_statuses_author_idx ON pulse_statuses (author_id, created_at DESC)
    WHERE closed_at IS NULL;

CREATE TABLE pulse_audience (
    status_id bigint NOT NULL REFERENCES pulse_statuses(id) ON DELETE CASCADE,
    -- Either a concrete user or a community; both columns are never set at once.
    user_id   bigint REFERENCES users(id) ON DELETE CASCADE,
    community_id bigint REFERENCES social_communities(id) ON DELETE CASCADE,
    PRIMARY KEY (status_id, user_id, community_id),
    CHECK ((user_id IS NULL) <> (community_id IS NULL))
);

CREATE TABLE pulse_responses (
    status_id  bigint      NOT NULL REFERENCES pulse_statuses(id) ON DELETE CASCADE,
    user_id    bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (status_id, user_id)
);
