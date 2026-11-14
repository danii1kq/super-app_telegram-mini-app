CREATE TABLE chat_attachments (
    message_id bigint PRIMARY KEY REFERENCES chat_messages(id) ON DELETE CASCADE,
    sender_id bigint NOT NULL REFERENCES users(id),
    byte_size bigint NOT NULL CHECK (byte_size > 0 AND byte_size <= 20971520),
    body_sealed bytea NOT NULL,
    key_id text NOT NULL
);
CREATE INDEX chat_attachments_sender_idx ON chat_attachments(sender_id);
