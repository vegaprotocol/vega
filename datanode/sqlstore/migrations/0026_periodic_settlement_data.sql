-- +goose Up

ALTER TYPE proposal_error ADD VALUE IF NOT EXISTS 'PROPOSAL_ERROR_INVALID_PERPETUAL_PRODUCT';

CREATE TABLE IF NOT EXISTS funding_period (
    market_id BYTEA NOT NULL,
    funding_period_seq BIGINT NOT NULL,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE,
    funding_payment NUMERIC,
    funding_rate NUMERIC,
    external_twap NUMERIC,
    internal_twap NUMERIC,
    vega_time TIMESTAMP WITH TIME ZONE NOT NULL,
    tx_hash BYTEA NOT NULL,
    PRIMARY KEY (market_id, funding_period_seq)
);

CREATE TYPE funding_period_data_point_source AS ENUM ('SOURCE_UNSPECIFIED', 'SOURCE_EXTERNAL', 'SOURCE_INTERNAL');

CREATE TABLE IF NOT EXISTS funding_period_data_points (
    market_id BYTEA NOT NULL,
    funding_period_seq BIGINT NOT NULL,
    data_point_type funding_period_data_point_source NOT NULL,
    price NUMERIC NOT NULL,
    twap NUMERIC NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    vega_time TIMESTAMP WITH TIME ZONE NOT NULL,
    tx_hash BYTEA NOT NULL,
    PRIMARY KEY (market_id, funding_period_seq, data_point_type, vega_time),
    -- Because we really shouldn't have a funding period data point for a non-existent funding period.
    FOREIGN KEY (market_id, funding_period_seq) REFERENCES funding_period(market_id, funding_period_seq)
);

-- +goose Down

DROP TABLE IF EXISTS funding_period_data_points cascade;
DROP TYPE IF EXISTS funding_period_data_point_source;
DROP TABLE IF EXISTS funding_period cascade;
