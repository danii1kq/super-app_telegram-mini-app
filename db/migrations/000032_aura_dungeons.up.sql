ALTER TABLE aura_players ADD COLUMN progress jsonb NOT NULL DEFAULT '{}'::jsonb;
CREATE TABLE aura_weekly_scores (
    week date NOT NULL,
    user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    depth integer NOT NULL,
    difficulty integer NOT NULL,
    errors integer NOT NULL,
    clears integer NOT NULL DEFAULT 1,
    PRIMARY KEY (week,user_id)
);
