-- Default time zone for new accounts: Asia/Almaty.
--
-- Existing rows still on the old default are moved too: the app never offered a
-- time zone picker, so a stored Asia/Qyzylorda was the default, not a choice.

ALTER TABLE user_settings ALTER COLUMN timezone SET DEFAULT 'Asia/Almaty';
UPDATE user_settings SET timezone = 'Asia/Almaty' WHERE timezone = 'Asia/Qyzylorda';
