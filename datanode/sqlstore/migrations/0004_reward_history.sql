-- +goose Up

ALTER TABLE rewards DROP CONSTRAINT IF EXISTS rewards_party_id_fkey;

SELECT create_hypertable('rewards', 'vega_time', chunk_time_interval => INTERVAL '1 day', migrate_data => true);
