DROP TABLE split_reminder_agreements;
ALTER TABLE notification_delivery DROP COLUMN reminder_version,DROP COLUMN lease_until,DROP COLUMN claim_token;
ALTER TABLE user_settings DROP COLUMN bot_notifications_opt_in;
