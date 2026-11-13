ALTER TABLE tinder_profiles DROP CONSTRAINT IF EXISTS tinder_profiles_photo_complete;
ALTER TABLE tinder_profiles
    DROP COLUMN IF EXISTS photo_sealed,
    DROP COLUMN IF EXISTS photo_key_id,
    DROP COLUMN IF EXISTS photo_mime;
