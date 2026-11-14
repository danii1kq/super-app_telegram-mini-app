DROP TABLE split_receipt_invites, split_receipt_approvals;
UPDATE split_receipts SET status='draft' WHERE status IN ('collecting','review','cancelled');
ALTER TABLE split_receipts DROP COLUMN review_sealed, DROP COLUMN review_key_id, DROP CONSTRAINT split_receipts_status_check;
ALTER TABLE split_receipts ADD CONSTRAINT split_receipts_status_check CHECK(status IN ('draft','saved'));
