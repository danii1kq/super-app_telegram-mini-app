-- SruSplit: categories, receipt image lifecycle and the notification outbox.
-- Owner: backend/internal/apps/split (data) and backend/internal/core/notify (delivery).

-- A category on an expense and, when a receipt has mixed goods, on a line.
-- Old rows stay NULL and are reported as "Прочее" instead of disappearing.
ALTER TABLE split_expenses ADD COLUMN category text
    CHECK (category IS NULL OR category IN ('groceries', 'cafe', 'transport', 'home', 'fun', 'other'));
ALTER TABLE split_receipt_items ADD COLUMN category text
    CHECK (category IS NULL OR category IN ('groceries', 'cafe', 'transport', 'home', 'fun', 'other'));

-- The report reads by group, date and payer; this index keeps it off a seq scan.
CREATE INDEX split_expenses_report_idx
    ON split_expenses (group_id, spent_at DESC, id) WHERE deleted_at IS NULL;

-- Image lifecycle.
--
-- image_state says what the stored bytes are:
--   none      a receipt typed by hand, no photo at all
--   stored    an optimized image is present
--   purged    the photo was deleted by the retention policy; the receipt,
--             its items, shares, expenses and debts all stay
--   processing a worker is reading it right now, so cleanup must skip it
ALTER TABLE split_receipts ADD COLUMN image_state text NOT NULL DEFAULT 'stored'
    CHECK (image_state IN ('none', 'stored', 'purged', 'processing'));
-- When the photo becomes eligible for deletion. NULL means "not scheduled".
ALTER TABLE split_receipts ADD COLUMN image_expires_at timestamptz;
-- What was actually stored, after normalization, not what was uploaded.
ALTER TABLE split_receipts ADD COLUMN image_width integer;
ALTER TABLE split_receipts ADD COLUMN image_height integer;

CREATE INDEX split_receipts_purge_idx
    ON split_receipts (image_expires_at)
    WHERE image_state = 'stored';

-- Notification outbox.
--
-- One row per event that must reach a person. The payload is sealed with the
-- delivery key: the worker can send it but cannot read every expense in the app.
CREATE TABLE notification_delivery (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    -- The person who receives it.
    user_id        bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- Stable event name, e.g. split.debt_created.
    kind           text        NOT NULL,
    -- Sealed JSON: {text, button_url}. Never logged.
    payload_sealed bytea       NOT NULL,
    payload_key_id text        NOT NULL,
    -- Deduplication: the same event for the same person is one message.
    dedupe_key     text        NOT NULL,
    -- A newer change to the same subject supersedes an unsent older one.
    subject        text,
    subject_version bigint,
    state          text        NOT NULL DEFAULT 'pending'
                   CHECK (state IN ('pending', 'sent', 'skipped', 'failed')),
    attempts       integer     NOT NULL DEFAULT 0,
    -- Set when Telegram asked to slow down or the send failed temporarily.
    next_attempt_at timestamptz NOT NULL DEFAULT now(),
    last_error     text,
    sent_at        timestamptz,
    created_at     timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, dedupe_key)
);

CREATE INDEX notification_delivery_pending_idx
    ON notification_delivery (next_attempt_at, id) WHERE state = 'pending';
CREATE INDEX notification_delivery_subject_idx
    ON notification_delivery (subject, user_id) WHERE state = 'pending';

-- Per-person notification settings for this category.
--
-- The general rule hides amounts; SruSplit debts are the exception, so a person
-- can turn the category off or hide the amounts in it.
ALTER TABLE user_settings ADD COLUMN split_debt_notifications boolean NOT NULL DEFAULT true;
ALTER TABLE user_settings ADD COLUMN split_debt_show_amounts boolean NOT NULL DEFAULT true;
