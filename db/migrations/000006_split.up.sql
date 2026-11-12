-- SruSplit: expense groups, expenses and shares. Owner: backend/internal/apps/split.

CREATE TABLE split_groups (
    id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    owner_id     bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title_sealed bytea       NOT NULL,
    key_id       text        NOT NULL,
    currency     text        NOT NULL DEFAULT 'KZT' CHECK (currency = 'KZT'),
    created_at   timestamptz NOT NULL DEFAULT now(),
    archived_at  timestamptz
);

CREATE TABLE split_members (
    group_id  bigint      NOT NULL REFERENCES split_groups(id) ON DELETE CASCADE,
    user_id   bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    state     text        NOT NULL DEFAULT 'active'
              CHECK (state IN ('active', 'left')),
    joined_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (group_id, user_id)
);

CREATE INDEX split_members_user_idx ON split_members (user_id) WHERE state = 'active';

CREATE TABLE split_expenses (
    id                bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    group_id          bigint      NOT NULL REFERENCES split_groups(id) ON DELETE CASCADE,
    author_id         bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    payer_id          bigint      NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    description_sealed bytea      NOT NULL,
    amount_sealed     bytea       NOT NULL,
    key_id            text        NOT NULL,
    client_request_id text        NOT NULL,
    version           integer     NOT NULL DEFAULT 1,
    spent_at          timestamptz NOT NULL DEFAULT now(),
    created_at        timestamptz NOT NULL DEFAULT now(),
    deleted_at        timestamptz,
    UNIQUE (author_id, client_request_id)
);

CREATE INDEX split_expenses_group_idx ON split_expenses (group_id, spent_at, id)
    WHERE deleted_at IS NULL;

CREATE TABLE split_shares (
    expense_id   bigint NOT NULL REFERENCES split_expenses(id) ON DELETE CASCADE,
    user_id      bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    amount_sealed bytea NOT NULL,
    key_id       text   NOT NULL,
    PRIMARY KEY (expense_id, user_id)
);
