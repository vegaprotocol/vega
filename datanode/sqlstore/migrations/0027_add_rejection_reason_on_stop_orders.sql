-- +goose Up

-- +goose StatementBegin
DO
$$
BEGIN
    IF NOT EXISTS (SELECT * FROM pg_type typ JOIN pg_namespace ns ON ns.oid = typ.typnamespace
       WHERE ns.nspname = current_schema() AND typ.typname = 'stop_order_rejection_reason')
    THEN
        CREATE TYPE stop_order_rejection_reason as enum(
            'REJECTION_REASON_UNSPECIFIED',
            'REJECTION_REASON_TRADING_NOT_ALLOWED',
            'REJECTION_REASON_EXPIRY_IN_THE_PAST',
            'REJECTION_REASON_MUST_BE_REDUCE_ONLY',
            'REJECTION_REASON_MAX_STOP_ORDERS_PER_PARTY_REACHED',
            'REJECTION_REASON_STOP_ORDER_NOT_ALLOWED_WITHOUT_A_POSITION',
            'REJECTION_REASON_STOP_ORDER_NOT_CLOSING_THE_POSITION');
    END IF;
END
$$;
-- +goose StatementEnd

ALTER TABLE stop_orders ADD COLUMN IF NOT EXISTS rejection_reason stop_order_rejection_reason NOT NULL DEFAULT 'REJECTION_REASON_UNSPECIFIED';

-- +goose Down

-- Don't do anything, just leave the column there, the up script shouldn't add it if it already exists.
