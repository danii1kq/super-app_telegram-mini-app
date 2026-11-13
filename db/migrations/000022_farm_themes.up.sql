CREATE TABLE farm_theme_codes (
    code_hash text PRIMARY KEY,
    theme text NOT NULL CHECK (theme IN ('tropical','lunar')),
    max_uses integer NOT NULL CHECK (max_uses > 0),
    uses integer NOT NULL DEFAULT 0 CHECK (uses >= 0),
    expires_at timestamptz
);
CREATE TABLE farm_theme_redemptions (
    code_hash text NOT NULL REFERENCES farm_theme_codes(code_hash),
    user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    PRIMARY KEY (code_hash,user_id)
);
CREATE TABLE farm_theme_orders (
    payload_hash text PRIMARY KEY,
    user_id bigint NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    payer_lookup text NOT NULL,
    theme text NOT NULL CHECK (theme IN ('tropical','lunar')),
    amount integer NOT NULL CHECK (amount > 0),
    created_at timestamptz NOT NULL DEFAULT now(),
    paid_at timestamptz,
    charge_id text UNIQUE
);
