-- +goose Up
CREATE TABLE IF NOT EXISTS funding_period (
    market_id BYTEA NOT NULL,
    funding_period_seq BIGINT NOT NULL,
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE,
    funding_payment NUMERIC,
    funding_rate NUMERIC,
    vega_time TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY (market_id, funding_period_seq)
);

CREATE TYPE funding_period_data_point_source AS ENUM ('SOURCE_UNSPECIFIED', 'SOURCE_EXTERNAL', 'SOURCE_INTERNAL');

CREATE TABLE IF NOT EXISTS funding_period_data_points (
    market_id BYTEA NOT NULL,
    funding_period_seq BIGINT NOT NULL,
    data_point_type funding_period_data_point_source NOT NULL,
    price NUMERIC NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    vega_time TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY (market_id, funding_period_seq, data_point_type, vega_time)
);

-- +goose Down

DROP TABLE IF EXISTS funding_period_data_points;
DROP TYPE IF EXISTS funding_period_data_point_source;
DROP TABLE IF EXISTS funding_period;
