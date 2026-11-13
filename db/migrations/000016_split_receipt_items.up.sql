ALTER TABLE split_receipts ADD COLUMN version integer NOT NULL DEFAULT 1;
ALTER TABLE split_receipt_items ADD COLUMN allocation_mode text NOT NULL DEFAULT 'units' CHECK (allocation_mode IN ('units','shared'));
ALTER TABLE split_receipt_items ADD COLUMN parent_position integer;
ALTER TABLE split_receipt_items ADD COLUMN suggested_parent_position integer;
UPDATE split_receipt_items i SET allocation_mode='shared' WHERE
 (SELECT coalesce(sum(units),0) FROM split_receipt_item_units WHERE item_id=i.id)>i.quantity;
