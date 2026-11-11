-- SruSplit: receipts with items, item-level allocation, debts and repayments.
-- Owner: backend/internal/apps/split.
--
-- One rule keeps the money consistent: an expense defines the obligation, a
-- repayment settles it. A debt row never adds a second amount on top of an
-- expense it points at.

-- A photographed receipt. The image is sealed and readable only by the group.
CREATE TABLE split_receipts (
    id             bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    group_id       bigint      NOT NULL REFERENCES split_groups(id) ON DELETE CASCADE,
    author_id      bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    image_sealed   bytea       NOT NULL,
    image_key_id   text        NOT NULL,
    image_mime     text        NOT NULL CHECK (image_mime IN ('image/jpeg', 'image/png', 'image/webp')),
    -- Zero means a receipt typed by hand: recognition may be unavailable, and a
    -- paper receipt may simply be lost.
    image_bytes    integer     NOT NULL CHECK (image_bytes >= 0 AND image_bytes <= 5242880),
    -- draft: recognized, editable, not yet an expense. saved: committed.
    status         text        NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'saved')),
    -- Why recognition failed, as a stable code the UI translates. Never the
    -- provider's raw message: that can carry the key or the image.
    ocr_error      text,
    -- Totals as printed on the receipt, in minor units. Null means unreadable.
    total_sealed   bytea,
    total_key_id   text,
    discount_sealed bytea,
    discount_key_id text,
    service_sealed bytea,
    service_key_id text,
    saved_expense_id bigint    REFERENCES split_expenses(id) ON DELETE SET NULL,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX split_receipts_group_idx ON split_receipts (group_id, created_at DESC);

-- One line of the receipt. amount_minor is the price of the whole line.
CREATE TABLE split_receipt_items (
    id            bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    receipt_id    bigint  NOT NULL REFERENCES split_receipts(id) ON DELETE CASCADE,
    position      integer NOT NULL,
    name_sealed   bytea   NOT NULL,
    name_key_id   text    NOT NULL,
    quantity      integer NOT NULL DEFAULT 1 CHECK (quantity > 0 AND quantity <= 999),
    amount_sealed bytea   NOT NULL,
    amount_key_id text    NOT NULL,
    -- unclear: the recognizer could not read it. The UI highlights these
    -- instead of guessing a value.
    confidence    text    NOT NULL DEFAULT 'ok' CHECK (confidence IN ('ok', 'unclear')),
    UNIQUE (receipt_id, position)
);

-- How many units of one line belong to one person.
--
-- Units, not percentages: three drinks can go two to one person and one to
-- another, and the sum of units per line must equal the line quantity so a unit
-- is never counted twice.
CREATE TABLE split_receipt_item_units (
    item_id bigint  NOT NULL REFERENCES split_receipt_items(id) ON DELETE CASCADE,
    user_id bigint  NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    units   integer NOT NULL CHECK (units > 0 AND units <= 999),
    PRIMARY KEY (item_id, user_id)
);

-- An obligation between two people.
--
-- source_expense_id set: this row points at an obligation the expense already
-- created. It is a handle for repayments and never adds to the balance.
-- source_expense_id null: a manual debt with no expense behind it, and it does
-- add to the balance.
CREATE TABLE split_debts (
    id                bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    group_id          bigint      NOT NULL REFERENCES split_groups(id) ON DELETE CASCADE,
    author_id         bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    creditor_id       bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    debtor_id         bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount_sealed     bytea       NOT NULL,
    amount_key_id     text        NOT NULL,
    note_sealed       bytea,
    note_key_id       text,
    source_expense_id bigint      REFERENCES split_expenses(id) ON DELETE CASCADE,
    client_request_id text        NOT NULL,
    created_at        timestamptz NOT NULL DEFAULT now(),
    CHECK (creditor_id <> debtor_id),
    UNIQUE (author_id, client_request_id)
);

CREATE INDEX split_debts_group_idx ON split_debts (group_id, created_at DESC);
-- One tracked obligation per expense and debtor: recording the same share twice
-- would show the same money as two debts.
CREATE UNIQUE INDEX split_debts_source_idx
    ON split_debts (source_expense_id, debtor_id) WHERE source_expense_id IS NOT NULL;

-- A repayment. Rows are append-only: the history is the record.
CREATE TABLE split_debt_payments (
    id                bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    debt_id           bigint      NOT NULL REFERENCES split_debts(id) ON DELETE CASCADE,
    author_id         bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount_sealed     bytea       NOT NULL,
    amount_key_id     text        NOT NULL,
    client_request_id text        NOT NULL,
    paid_at           timestamptz NOT NULL DEFAULT now(),
    UNIQUE (author_id, client_request_id)
);

CREATE INDEX split_debt_payments_debt_idx ON split_debt_payments (debt_id, paid_at);
