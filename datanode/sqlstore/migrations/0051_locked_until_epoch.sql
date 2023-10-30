-- +goose Up
ALTER TABLE rewards ADD COLUMN locked_until_epoch_id BIGINT;
UPDATE rewards SET locked_until_epoch_id = epoch_id;
ALTER TABLE rewards ALTER COLUMN locked_until_epoch_id SET NOT NULL;

-- +goose Down
ALTER TABLE rewards DROP COLUMN locked_until_epoch_id;
