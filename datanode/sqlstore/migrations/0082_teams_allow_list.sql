-- +goose Up
ALTER TABLE teams
  ADD COLUMN IF NOT EXISTS allow_list JSONB;

-- +goose Down

ALTER TABLE teams
  DROP COLUMN IF EXISTS allow_list;
