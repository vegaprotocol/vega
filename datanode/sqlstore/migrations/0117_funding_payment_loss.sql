-- +goose Up

ALTER TABLE funding_payment ADD COLUMN IF NOT EXISTS loss_socialisation_amount NUMERIC NOT NULL DEFAULT (0);

-- +goose Down

ALTER TABLE funding_payment DROP COLUMN IF EXISTS loss_socialisation_amount;
