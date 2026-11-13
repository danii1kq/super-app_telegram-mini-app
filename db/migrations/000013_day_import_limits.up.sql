CREATE TABLE day_import_limits (
    user_id bigint PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    usage_day date NOT NULL,
    used integer NOT NULL CHECK (used BETWEEN 1 AND 20),
    last_request_at timestamptz NOT NULL
);
