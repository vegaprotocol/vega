-- +goose Up

DROP TRIGGER IF EXISTS update_funding_payment ON funding_payment;

ALTER TABLE funding_payment ADD COLUMN IF NOT EXISTS loss_socialisation_amount NUMERIC NOT NULL DEFAULT (0);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_funding_payments()
       RETURNS TRIGGER
       language plpgsql
AS $$
   BEGIN
        INSERT INTO funding_payment(market_id, party_id, funding_period_seq, amount, vega_time, tx_hash, loss_socialisation_amount)
        VALUES (new.market_id, new.party_id, new.funding_period_seq, new.amount, new.vega_time, new.tx_hash, new.loss_socialisation_amount)
        ON CONFLICT(party_id, market_id, vega_time)
        DO UPDATE SET
            amount = excluded.amount,
            loss_socialisation_amount = excluded.loss_socialisation_amount;
        RETURN NULL;
    END;
$$;
-- +goose StatementEnd

CREATE TRIGGER update_funding_payment
    AFTER INSERT 
    ON funding_payment
    FOR EACH ROW EXECUTE FUNCTION update_funding_payments();

-- +goose Down

DROP TRIGGER IF EXISTS update_funding_payment ON funding_payment;

ALTER TABLE funding_payment DROP COLUMN IF EXISTS loss_socialisation_amount;

DROP FUNCTION IF EXISTS update_funding_payments();

