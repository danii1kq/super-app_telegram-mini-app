CREATE TABLE farm_players (
    user_id bigint PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    state jsonb NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE farm_weekly_sales (
    week date NOT NULL,
    user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    revenue bigint NOT NULL CHECK (revenue >= 0),
    PRIMARY KEY (week, user_id)
);
CREATE INDEX farm_weekly_ranking ON farm_weekly_sales (week, revenue DESC, user_id);
