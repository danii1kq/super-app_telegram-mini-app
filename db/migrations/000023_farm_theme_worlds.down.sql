ALTER TABLE farm_theme_codes DROP CONSTRAINT farm_theme_codes_theme_check;
ALTER TABLE farm_theme_codes ADD CONSTRAINT farm_theme_codes_theme_check CHECK (theme IN ('tropical','lunar'));
ALTER TABLE farm_theme_orders DROP CONSTRAINT farm_theme_orders_theme_check;
ALTER TABLE farm_theme_orders ADD CONSTRAINT farm_theme_orders_theme_check CHECK (theme IN ('tropical','lunar'));
