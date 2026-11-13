ALTER TABLE user_settings
    DROP COLUMN IF EXISTS tinder_like_notifications,
    DROP COLUMN IF EXISTS tinder_match_notifications;
