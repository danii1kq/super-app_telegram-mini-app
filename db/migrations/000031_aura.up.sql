-- Aura Arena: a math-driven RPG for SruGame.
-- Owner: backend/internal/apps/aura.
--
-- One row per player holds the whole hero: aura, level, stats, cooldowns and
-- the active session. The session lives in the row rather than in process
-- memory, so a restart mid-arena does not silently eat the attempt.
--
-- Relics are their own table because a player owns many and the deck needs to
-- query them by slot.

CREATE TABLE aura_players (
    user_id        bigint PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    -- The hero path: strategist, berserker, sage, keeper or hunter.
    hero_class     text        NOT NULL DEFAULT ''
                   CHECK (hero_class IN ('', 'strategist', 'berserker', 'sage', 'keeper', 'hunter')),
    aura           bigint      NOT NULL DEFAULT 0 CHECK (aura >= 0),
    total_aura     bigint      NOT NULL DEFAULT 0,
    level          integer     NOT NULL DEFAULT 1 CHECK (level >= 1),
    xp             integer     NOT NULL DEFAULT 0,
    stat_points    integer     NOT NULL DEFAULT 3,
    stat_str       integer     NOT NULL DEFAULT 0,
    stat_agi       integer     NOT NULL DEFAULT 0,
    stat_int       integer     NOT NULL DEFAULT 0,
    stat_vit       integer     NOT NULL DEFAULT 0,
    stat_per       integer     NOT NULL DEFAULT 0,
    -- Cooldowns are absolute times, so the server clock decides, not the client.
    farm_ready_at  timestamptz,
    arena_ready_at timestamptz,
    gate_ready_at  timestamptz,
    tower_ready_at timestamptz,
    -- Daily grind cap: the counter resets when the UTC day changes.
    grind_day      date,
    grind_today    integer     NOT NULL DEFAULT 0,
    -- The one open challenge, if any. Cleared when it resolves or expires.
    session        jsonb       NOT NULL DEFAULT '{}'::jsonb,
    streak         integer     NOT NULL DEFAULT 0,
    last_seen_day  date,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);

-- The leaderboard reads by aura, so the index keeps that cheap.
CREATE INDEX aura_players_aura_idx ON aura_players (aura DESC, user_id);

CREATE TABLE aura_relics (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    relic_key  text        NOT NULL,
    rank       text        NOT NULL
               CHECK (rank IN ('E', 'D', 'C', 'B', 'A', 'S', 'SS', 'SSS')),
    level      integer     NOT NULL DEFAULT 1 CHECK (level BETWEEN 1 AND 5),
    -- One relic per slot may be worn; empty means it sits in the bag.
    slot       text        NOT NULL DEFAULT ''
               CHECK (slot IN ('', 'weapon', 'armor', 'accessory')),
    obtained_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, relic_key)
);

CREATE INDEX aura_relics_user_idx ON aura_relics (user_id, rank);

-- A worn relic is unique per slot, so two weapons cannot both count.
CREATE UNIQUE INDEX aura_relics_one_per_slot ON aura_relics (user_id, slot)
    WHERE slot <> '';

-- Every finished challenge is recorded once. The request id makes a retried
-- submit a no-op instead of a second reward.
CREATE TABLE aura_runs (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id    bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    kind       text        NOT NULL
               CHECK (kind IN ('farm', 'arena', 'gate', 'grind', 'tower')),
    request_id text        NOT NULL,
    aura_gain  bigint      NOT NULL DEFAULT 0,
    xp_gain    integer     NOT NULL DEFAULT 0,
    damage     integer     NOT NULL DEFAULT 0,
    correct    integer     NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, request_id)
);

CREATE INDEX aura_runs_user_idx ON aura_runs (user_id, created_at DESC);
