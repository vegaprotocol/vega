-- +goose Up

ALTER TABLE amms ADD COLUMN IF NOT EXISTS proposed_fee numeric;

-- +goose Down

ALTER TABLE amms DROP COLUMN IF EXISTS proposed_fee;
