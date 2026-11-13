DO $$ BEGIN IF EXISTS(SELECT 1 FROM split_debt_payments WHERE status<>'confirmed') THEN RAISE EXCEPTION 'Resolve pending/rejected payments before downgrade'; END IF; END $$;
ALTER TABLE split_debt_payments DROP COLUMN status, DROP COLUMN method;
ALTER TABLE split_debts DROP COLUMN version;
ALTER TABLE split_groups DROP COLUMN financial_version;
