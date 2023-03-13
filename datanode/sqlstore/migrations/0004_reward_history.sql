-- +goose Up

ALTER TABLE rewards DROP CONSTRAINT IF EXISTS rewards_party_id_fkey;

SELECT create_hypertable('rewards', 'vega_time', chunk_time_interval => INTERVAL '1 day', if_not_exists => true, migrate_data => true);

-- +goose Down

-- We need to restore rewards as a regular table, while still preserving the data
-- unfortunately, due to the nature of hypertables, we have to do this by migrating
-- the data to a new table, dropping the hypertable, and then renaming the table
CREATE TABLE rewards_backup(
    party_id         BYTEA NOT NULL REFERENCES parties(id),
    asset_id         BYTEA NOT NULL,
    market_id        BYTEA NOT NULL,
    reward_type      TEXT NOT NULL,
    epoch_id         BIGINT NOT NULL,
    amount           HUGEINT,
    percent_of_total FLOAT,
    timestamp        TIMESTAMP WITH TIME ZONE NOT NULL,
    tx_hash          BYTEA NOT NULL,
    vega_time        TIMESTAMP WITH TIME ZONE NOT NULL,
    seq_num           BIGINT NOT NULL,
    primary key (vega_time, seq_num)
);

INSERT INTO rewards_backup
SELECT *
FROM rewards;

DROP TABLE rewards;

ALTER TABLE rewards_backup RENAME TO rewards;

CREATE INDEX ON rewards (party_id, asset_id);
CREATE INDEX ON rewards (asset_id);
CREATE INDEX ON rewards (epoch_id);
