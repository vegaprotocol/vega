-- +goose Up

ALTER TYPE stop_order_rejection_reason ADD VALUE IF NOT EXISTS 'REJECTION_REASON_STOP_ORDER_SIZE_OVERRIDE_UNSUPPORTED_FOR_SPOT';

-- +goose Down

-- Do nothing, if it already exists it won't matter and won't be recreated by the up migration.
