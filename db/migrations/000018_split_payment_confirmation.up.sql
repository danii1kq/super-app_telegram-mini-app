ALTER TABLE split_groups ADD COLUMN financial_version bigint NOT NULL DEFAULT 1;
ALTER TABLE split_debts ADD COLUMN version bigint NOT NULL DEFAULT 1;
ALTER TABLE split_debt_payments ADD COLUMN status text NOT NULL DEFAULT 'confirmed' CHECK(status IN ('pending','confirmed','rejected'));
ALTER TABLE split_debt_payments ADD COLUMN method text NOT NULL DEFAULT 'transfer' CHECK(method IN ('transfer','offset'));
CREATE INDEX split_debt_payments_pending_idx ON split_debt_payments(debt_id) WHERE status='pending';
