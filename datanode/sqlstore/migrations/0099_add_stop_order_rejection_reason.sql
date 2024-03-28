-- +goose Up

ALTER TYPE stop_order_rejection_reason ADD VALUE IF NOT EXISTS 'REJECTION_REASON_STOP_ORDER_LINKED_PERCENTAGE_INVALID' AFTER 'REJECTION_REASON_STOP_ORDER_NOT_CLOSING_THE_POSITION';

-- +goose Down

-- Do nothing, if it already exists it won't matter and won't be recreated by the up migration.
