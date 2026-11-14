ALTER TABLE user_settings ADD COLUMN bot_notifications_opt_in boolean NOT NULL DEFAULT false;
ALTER TABLE notification_delivery ADD COLUMN claim_token text;
ALTER TABLE notification_delivery ADD COLUMN lease_until timestamptz;
ALTER TABLE notification_delivery ADD COLUMN reminder_version integer;
CREATE TABLE split_reminder_agreements(
 debt_id bigint PRIMARY KEY REFERENCES split_debts(id) ON DELETE CASCADE,
 proposer_id bigint NOT NULL REFERENCES users(id),
 version integer NOT NULL DEFAULT 1,
 deadline_at timestamptz NOT NULL,
 timezone text NOT NULL,
 interval_days integer NOT NULL CHECK(interval_days IN (1,3,7)),
 status text NOT NULL DEFAULT 'proposed' CHECK(status IN ('proposed','active','disabled')),
 next_run_at timestamptz NOT NULL,
 updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX split_reminder_due_idx ON split_reminder_agreements(next_run_at,debt_id) WHERE status='active';
