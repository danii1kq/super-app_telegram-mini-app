-- SruTinder: profiles, likes and matches.
-- Owner: backend/internal/apps/tinder.
--
-- A profile is one row per account. The bio is sealed like every other piece
-- of user text in this project, because it is free-form and personal.
--
-- A like is directional and stored once. A match exists when both directions
-- are present; it is materialised as its own row so the chat can hang off it
-- and so a match is created exactly once, even under concurrent likes.

CREATE TABLE tinder_profiles (
    user_id      bigint PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    bio_sealed   bytea       NOT NULL,
    key_id       text        NOT NULL,
    photo_url    text        NOT NULL DEFAULT '',
    city         text        NOT NULL DEFAULT '',
    age          integer     CHECK (age IS NULL OR (age >= 18 AND age <= 120)),
    active       boolean     NOT NULL DEFAULT true,
    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now()
);

-- The deck reads active profiles newest-first; the index keeps that cheap.
CREATE INDEX tinder_profiles_active_idx ON tinder_profiles (updated_at DESC)
    WHERE active;

CREATE TABLE tinder_likes (
    from_id    bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    to_id      bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (from_id, to_id),
    CHECK (from_id <> to_id)
);

-- "Who liked me" and the mutual check both read by recipient.
CREATE INDEX tinder_likes_to_idx ON tinder_likes (to_id, created_at DESC);

CREATE TABLE tinder_matches (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_a     bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_b     bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    -- Storing the pair ordered makes the uniqueness constraint possible.
    CHECK (user_a < user_b),
    UNIQUE (user_a, user_b)
);

CREATE INDEX tinder_matches_a_idx ON tinder_matches (user_a, created_at DESC);
CREATE INDEX tinder_matches_b_idx ON tinder_matches (user_b, created_at DESC);

-- A pass is a decision not to see this profile again. It is kept separate from
-- a like so the deck can exclude both without confusing the two.
CREATE TABLE tinder_passes (
    from_id    bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    to_id      bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (from_id, to_id),
    CHECK (from_id <> to_id)
);
