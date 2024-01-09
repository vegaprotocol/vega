-- +goose Up

ALTER TYPE stop_order_rejection_reason ADD VALUE IF NOT EXISTS 'REJECTION_REASON_STOP_ORDER_CANNOT_MATCH_OCO_EXPIRY_TIMES';

-- +goose Down

-- Do nothing, if it already exists it won't matter and won't be recreated by the up migration.
