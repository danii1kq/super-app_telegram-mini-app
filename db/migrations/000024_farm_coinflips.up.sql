CREATE TABLE farm_coinflips (
 id text PRIMARY KEY,
 host_id bigint NOT NULL REFERENCES users(id),
 guest_id bigint REFERENCES users(id),
 stake bigint NOT NULL CHECK (stake BETWEEN 100 AND 100000),
 host_side text NOT NULL CHECK (host_side IN ('heads','tails')),
 result_side text NOT NULL DEFAULT '',
 winner_id bigint REFERENCES users(id),
 status text NOT NULL CHECK (status IN ('open','done','cancelled','expired')),
 created_at timestamptz NOT NULL,
 expires_at timestamptz NOT NULL,
 resolved_at timestamptz
);
CREATE UNIQUE INDEX farm_coinflips_one_open ON farm_coinflips(host_id) WHERE status='open';
CREATE INDEX farm_coinflips_lobby ON farm_coinflips(status,created_at DESC);
