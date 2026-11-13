-- SruTinder: gender and who the viewer is looking for.
-- Owner: backend/internal/apps/tinder.
--
-- Both fields are optional so an existing profile stays valid: a row without a
-- gender simply never matches a preference filter, it does not break the deck.
--
-- The preference is stored as an array because "looking for" is not always one
-- value. An empty array means "no preference", which is the default and shows
-- everyone, so a player who does not care is never filtered out of anyone's deck.

ALTER TABLE tinder_profiles
    ADD COLUMN gender text NOT NULL DEFAULT ''
        CHECK (gender IN ('', 'male', 'female', 'other'));

ALTER TABLE tinder_profiles
    ADD COLUMN seeking text[] NOT NULL DEFAULT '{}'
        CHECK (seeking <@ ARRAY['male','female','other']::text[]);

-- The deck filters on both sides of the pair, so the planner needs the gender
-- of a candidate cheaply. The partial index keeps it to complete profiles.
CREATE INDEX tinder_profiles_gender_idx ON tinder_profiles (gender)
    WHERE active AND gender <> '';
