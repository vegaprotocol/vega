-- +goose Up

ALTER TABLE parties
  ADD COLUMN IF NOT EXISTS alias VARCHAR(32) NOT NULL DEFAULT '';

ALTER TABLE parties
  ADD COLUMN IF NOT EXISTS metadata JSONB;

UPDATE parties SET alias = 'network' WHERE id = '\x03';

-- +goose Down

ALTER TABLE parties
  DROP COLUMN IF EXISTS alias;

ALTER TABLE parties
  DROP COLUMN IF EXISTS metadata;
