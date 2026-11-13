ALTER TABLE split_receipts DROP CONSTRAINT split_receipts_status_check;
ALTER TABLE split_receipts ADD CONSTRAINT split_receipts_status_check CHECK(status IN ('draft','collecting','review','saved','cancelled'));
ALTER TABLE split_receipts ADD COLUMN review_sealed bytea;
ALTER TABLE split_receipts ADD COLUMN review_key_id text;
CREATE TABLE split_receipt_approvals(
 receipt_id bigint NOT NULL REFERENCES split_receipts(id) ON DELETE CASCADE,
 user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 revision integer NOT NULL,
 decision text NOT NULL DEFAULT 'pending' CHECK(decision IN ('pending','accepted','question')),
 question_sealed bytea, question_key_id text,
 PRIMARY KEY(receipt_id,user_id)
);
CREATE TABLE split_receipt_invites(
 token_hash text PRIMARY KEY,
 receipt_id bigint NOT NULL REFERENCES split_receipts(id) ON DELETE CASCADE,
 author_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
 expires_at timestamptz NOT NULL DEFAULT now()+interval '7 days',
 revoked_at timestamptz,
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX split_receipt_invites_receipt_idx ON split_receipt_invites(receipt_id);
