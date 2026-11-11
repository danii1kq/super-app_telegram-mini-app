-- Default time zone for new accounts: Asia/Almaty.
-- Existing rows keep whatever they already have: a person may have chosen one.

ALTER TABLE user_settings ALTER COLUMN timezone SET DEFAULT 'Asia/Almaty';
