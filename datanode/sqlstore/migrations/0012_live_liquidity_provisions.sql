-- +goose Up
DROP TRIGGER IF EXISTS update_current_liquidity_provisions on liquidity_provisions;
DROP TABLE IF EXISTS current_liquidity_provisions;
DROP FUNCTION IF EXISTS update_current_liquidity_provisions;


CREATE TABLE live_liquidity_provisions
(
    id BYTEA NOT NULL,
    party_id BYTEA,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    market_id BYTEA,
    commitment_amount HUGEINT,
    fee NUMERIC(1000, 16),
    sells jsonb,
    buys jsonb,
    version BIGINT,
    status liquidity_provision_status NOT NULL,
    reference TEXT,
    tx_hash BYTEA NOT NULL,
    vega_time TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY (id, vega_time)
);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_live_liquidity_provisions()
   RETURNS TRIGGER
   LANGUAGE PLPGSQL AS
$$
BEGIN

DELETE FROM live_liquidity_provisions
WHERE id = NEW.id;

-- We take into consideration Liquidity provisions with statuses:
-- Active (1), Undeployed (5), Pending (6)
IF NEW.status IN('STATUS_ACTIVE', 'STATUS_UNDEPLOYED', 'STATUS_PENDING')
    THEN
        INSERT INTO live_liquidity_provisions(id, party_id, created_at, updated_at,
                    market_id, commitment_amount, fee, sells, buys, version, status, reference, tx_hash, vega_time)
        VALUES(NEW.id, NEW.party_id, NEW.created_at, NEW.updated_at,
                NEW.market_id, NEW.commitment_amount, NEW.fee, NEW.sells,
                NEW.buys, NEW.version, NEW.status, NEW.reference, NEW.tx_hash, NEW.vega_time);
END IF;

RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER update_live_liquidity_provisions AFTER INSERT ON liquidity_provisions
    FOR EACH ROW EXECUTE FUNCTION update_live_liquidity_provisions();

-- +goose Down
DROP TRIGGER update_live_liquidity_provisions ON liquidity_provisions;
DROP FUNCTION update_live_liquidity_provisions;
DROP TABLE live_liquidity_provisions;
