-- Store the verified Telegram payload with the session so the trusted hop to
-- SruDoc and SruDay can reuse it while the session lives.

ALTER TABLE sessions
    ADD COLUMN init_data_sealed bytea,
    ADD COLUMN init_data_key_id text;
