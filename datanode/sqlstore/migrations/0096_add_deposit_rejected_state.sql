-- +goose Up

ALTER TYPE deposit_status ADD VALUE IF NOT EXISTS 'STATUS_DUPLICATE_REJECTED' AFTER 'STATUS_FINALIZED';

-- +goose Down

-- nothing to do
