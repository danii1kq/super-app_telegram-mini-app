CREATE INDEX split_receipts_recognition_budget_idx ON split_receipts(author_id, created_at DESC) WHERE image_bytes > 0;
