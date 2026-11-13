-- SruTinder: store the profile photo instead of pointing at a URL.
-- Owner: backend/internal/apps/tinder.
--
-- The photo is sealed like every other piece of user content in this project:
-- it is personal, and a public URL would leak it to anyone who guessed the
-- path. Serving goes through an authenticated route, not a static file.

ALTER TABLE tinder_profiles
    ADD COLUMN photo_sealed bytea NOT NULL DEFAULT ''::bytea,
    ADD COLUMN photo_key_id text NOT NULL DEFAULT '',
    ADD COLUMN photo_mime   text NOT NULL DEFAULT '';

-- A profile either has a stored photo or it does not; a half-written row would
-- mean a broken image for everyone who sees the card.
ALTER TABLE tinder_profiles
    ADD CONSTRAINT tinder_profiles_photo_complete CHECK (
        (photo_key_id = '' AND photo_mime = '')
        OR (photo_key_id <> '' AND photo_mime <> '')
    );
