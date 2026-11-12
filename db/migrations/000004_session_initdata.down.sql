ALTER TABLE sessions
    DROP COLUMN IF EXISTS init_data_sealed,
    DROP COLUMN IF EXISTS init_data_key_id;
