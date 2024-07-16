-- +goose Up

ALTER TABLE amms ADD COLUMN IF NOT EXISTS lower_virtual_liquidity numeric,
                 ADD COLUMN IF NOT EXISTS upper_virtual_liquidity numeric,
                 ADD COLUMN IF NOT EXISTS lower_theoretical_position numeric,
                 ADD COLUMN IF NOT EXISTS upper_theoretical_position numeric;

-- +goose Down

ALTER TABLE amms DROP COLUMN IF EXISTS lower_virtual_liquidity,
                 DROP COLUMN IF EXISTS upper_virtual_liquidity,
                 DROP COLUMN IF EXISTS lower_theoretical_position,
                 DROP COLUMN IF EXISTS upper_theoretical_position;
