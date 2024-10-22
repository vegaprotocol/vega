-- +goose Up

ALTER TYPE amm_status ADD VALUE IF NOT EXISTS 'STATUS_PENDING';

ALTER TABLE amms ADD COLUMN IF NOT EXISTS data_source_id bytea;
ALTER TABLE amms ADD COLUMN IF NOT EXISTS minimum_price_change_trigger numeric;

-- +goose Down

ALTER TABLE amms DROP COLUMN IF EXISTS data_source_id;
ALTER TABLE amms DROP COLUMN IF EXISTS minimum_price_change_trigger;
