CREATE TABLE split_settlements(
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 group_id bigint NOT NULL REFERENCES split_groups(id) ON DELETE CASCADE,
 author_id bigint NOT NULL REFERENCES users(id),
 user_a bigint NOT NULL REFERENCES users(id), user_b bigint NOT NULL REFERENCES users(id),
 group_version bigint NOT NULL,
 snapshot_sealed bytea NOT NULL, snapshot_key_id text NOT NULL,
 a_accepted boolean NOT NULL DEFAULT false, b_accepted boolean NOT NULL DEFAULT false,
 status text NOT NULL DEFAULT 'proposed' CHECK(status IN ('proposed','agreed','reported','confirmed','rejected','stale')),
 client_request_id text NOT NULL,
 created_at timestamptz NOT NULL DEFAULT now(),
 CHECK(user_a<user_b), UNIQUE(author_id,client_request_id)
);
CREATE INDEX split_settlements_group_idx ON split_settlements(group_id,id DESC);
CREATE UNIQUE INDEX split_settlements_active_idx ON split_settlements(group_id,user_a,user_b) WHERE status IN ('proposed','agreed','reported');
