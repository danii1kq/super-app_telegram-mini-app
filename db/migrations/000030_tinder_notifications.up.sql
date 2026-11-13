-- SruTinder: notification switches.
-- Owner: backend/internal/apps/tinder.
--
-- Two separate switches, because they are different events with different
-- privacy weight: a like is private to the person who sent it, a match is
-- mutual. Someone may want to know about a match and not about every like.
--
-- Both default to on: the point of the app is that something happens while you
-- are away, and an off-by-default notification is a notification nobody gets.

ALTER TABLE user_settings
    ADD COLUMN tinder_like_notifications boolean NOT NULL DEFAULT true;

ALTER TABLE user_settings
    ADD COLUMN tinder_match_notifications boolean NOT NULL DEFAULT true;
