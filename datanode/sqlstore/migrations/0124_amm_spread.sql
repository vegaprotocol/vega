-- +goose Up

ALTER TABLE amms ADD COLUMN IF NOT EXISTS spread bytea;

-- +goose Down

ALTER TABLE amms DROP COLUMN IF EXISTS spread;

