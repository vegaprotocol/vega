-- +goose Up

ALTER TABLE funding_period_data_points DROP CONSTRAINT IF EXISTS funding_period_data_points_market_id_funding_period_seq_fkey;

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT * FROM timescaledb_information.hypertables WHERE hypertable_name = 'funding_period_data_points') THEN
        PERFORM create_hypertable('funding_period_data_points','vega_time', chunk_time_interval => INTERVAL '1 day');
END IF;
END $$;
-- +goose StatementEnd

-- +goose Down

-- do nothing, we want funding_period_data_points to stay a hypertable regardless
