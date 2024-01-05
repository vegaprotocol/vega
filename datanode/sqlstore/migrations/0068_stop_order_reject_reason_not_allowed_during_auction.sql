-- +goose Up

ALTER TYPE stop_order_rejection_reason ADD VALUE IF NOT EXISTS 'REJECTION_REASON_STOP_ORDER_NOT_ALLOWED_DURING_OPENING_AUCTION';

-- +goose Down

-- Do nothing, if it already exists it won't matter and won't be recreated by the up migration.
