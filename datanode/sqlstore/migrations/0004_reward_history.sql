-- +goose Up

select create_hypertable('rewards', 'vega_time', chunk_time_interval => INTERVAL '1 day');
