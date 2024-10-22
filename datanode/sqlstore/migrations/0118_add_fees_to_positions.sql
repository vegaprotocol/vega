-- +goose Up

ALTER TABLE positions ADD COLUMN IF NOT EXISTS taker_fees_paid HUGEINT NOT NULL DEFAULT (0);
ALTER TABLE positions ADD COLUMN IF NOT EXISTS maker_fees_received HUGEINT NOT NULL DEFAULT (0);
ALTER TABLE positions ADD COLUMN IF NOT EXISTS fees_paid HUGEINT NOT NULL DEFAULT (0);
ALTER TABLE positions ADD COLUMN IF NOT EXISTS taker_fees_paid_since HUGEINT NOT NULL DEFAULT (0);
ALTER TABLE positions ADD COLUMN IF NOT EXISTS maker_fees_received_since HUGEINT NOT NULL DEFAULT (0);
ALTER TABLE positions ADD COLUMN IF NOT EXISTS fees_paid_since HUGEINT NOT NULL DEFAULT (0);
ALTER TABLE positions ADD COLUMN IF NOT EXISTS funding_payment_amount HUGEINT NOT NULL DEFAULT(0);
ALTER TABLE positions ADD COLUMN IF NOT EXISTS funding_payment_amount_since HUGEINT NOT NULL DEFAULT(0);

-- +goose Down

ALTER TABLE positions DROP COLUMN IF EXISTS taker_fees_paid;
ALTER TABLE positions DROP COLUMN IF EXISTS maker_fees_received;
ALTER TABLE positions DROP COLUMN IF EXISTS fees_paid;
ALTER TABLE positions DROP COLUMN IF EXISTS taker_fees_paid_since;
ALTER TABLE positions DROP COLUMN IF EXISTS maker_fees_received_since;
ALTER TABLE positions DROP COLUMN IF EXISTS fees_paid_since;
ALTER TABLE positions DROP COLUMN IF EXISTS funding_payment_amount;
ALTER TABLE positions DROP COLUMN IF EXISTS funding_payment_amount_since;
