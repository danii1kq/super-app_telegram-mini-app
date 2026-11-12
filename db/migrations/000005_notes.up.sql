-- SruNotes: notes and read grants. Owner: backend/internal/apps/notes.

CREATE TABLE notes (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    owner_id      bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title_sealed  bytea,
    body_sealed   bytea       NOT NULL,
    key_id        text        NOT NULL,
    version       integer     NOT NULL DEFAULT 1,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),
    deleted_at    timestamptz
);

CREATE INDEX notes_owner_idx ON notes (owner_id, updated_at DESC, id DESC)
    WHERE deleted_at IS NULL;

CREATE TABLE note_grants (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    note_id      bigint      NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    recipient_id bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    granted_by   bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- A grant may come from a chat or a community; the membership is re-checked
    -- on read, so leaving the group closes the access.
    context_kind text        NOT NULL DEFAULT 'direct'
                 CHECK (context_kind IN ('direct', 'chat', 'community')),
    context_id   bigint,
    state        text        NOT NULL DEFAULT 'active'
                 CHECK (state IN ('active', 'revoked')),
    created_at   timestamptz NOT NULL DEFAULT now(),
    UNIQUE (note_id, recipient_id)
);

CREATE INDEX note_grants_recipient_idx ON note_grants (recipient_id, note_id)
    WHERE state = 'active';
