-- Chats: conversations, members and messages. Owner: backend/internal/core/chat.

CREATE TABLE chat_conversations (
    id                bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    kind              text        NOT NULL CHECK (kind IN ('direct', 'group')),
    title_sealed      bytea,
    key_id            text,
    community_id      bigint      REFERENCES social_communities(id) ON DELETE SET NULL,
    pair_key          text        UNIQUE,
    last_message_seq  bigint      NOT NULL DEFAULT 0,
    created_at        timestamptz NOT NULL DEFAULT now(),
    CHECK (kind <> 'group' OR (title_sealed IS NOT NULL AND key_id IS NOT NULL))
);

CREATE TABLE chat_members (
    conversation_id bigint      NOT NULL REFERENCES chat_conversations(id) ON DELETE CASCADE,
    user_id         bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role            text        NOT NULL DEFAULT 'member' CHECK (role IN ('owner', 'member')),
    request_state   text        NOT NULL DEFAULT 'accepted'
                    CHECK (request_state IN ('pending', 'accepted', 'declined', 'blocked')),
    last_read_seq   bigint      NOT NULL DEFAULT 0,
    muted           boolean     NOT NULL DEFAULT false,
    joined_at       timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (conversation_id, user_id)
);

CREATE INDEX chat_members_user_idx ON chat_members (user_id, conversation_id);

CREATE TABLE chat_messages (
    id                bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    conversation_id   bigint      NOT NULL REFERENCES chat_conversations(id) ON DELETE CASCADE,
    sender_id         bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    seq               bigint      NOT NULL,
    client_message_id text        NOT NULL,
    payload_sealed    bytea       NOT NULL,
    key_id            text        NOT NULL,
    created_at        timestamptz NOT NULL DEFAULT now(),
    UNIQUE (conversation_id, seq),
    UNIQUE (sender_id, client_message_id)
);

CREATE INDEX chat_messages_history_idx ON chat_messages (conversation_id, seq DESC);
