DROP INDEX IF EXISTS tinder_profiles_gender_idx;
ALTER TABLE tinder_profiles
    DROP COLUMN IF EXISTS seeking,
    DROP COLUMN IF EXISTS gender;
