-- +goose Up

ALTER TABLE orders
      ADD COLUMN post_only BOOLEAN DEFAULT false,
      ADD COLUMN reduce_only BOOLEAN DEFAULT false;

-- +goose Down

ALTER TABLE orders
      DROP COLUMN IF EXISTS post_only,
      DROP COLUMN IF EXISTS reduce_only;
