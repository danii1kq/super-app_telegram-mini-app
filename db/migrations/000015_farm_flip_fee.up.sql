-- Existing offers keep the commission that applied when their stake was taken.
ALTER TABLE farm_coinflips ADD COLUMN fee_percent integer NOT NULL DEFAULT 10 CHECK (fee_percent BETWEEN 0 AND 100);
