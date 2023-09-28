-- +goose Up

CREATE TABLE IF NOT EXISTS funding_payment (
    market_id BYTEA NOT NULL,
    party_id BYTEA NOT NULL,
    amount NUMERIC,
    funding_period_seq BIGINT NOT NULL,
    vega_time TIMESTAMP WITH TIME ZONE NOT NULL,
    tx_hash BYTEA NOT NULL,
    PRIMARY KEY (party_id, market_id, vega_time)
);

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT * FROM timescaledb_information.hypertables WHERE hypertable_name = 'funding_payment') THEN
        PERFORM create_hypertable('funding_payment','vega_time', chunk_time_interval => INTERVAL '1 day');
END IF;
END $$;
-- +goose StatementEnd

-- +goose Down

DROP TABLE IF EXISTS funding_payment cascade;
