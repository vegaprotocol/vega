-- +goose Up

CREATE TABLE party_activity_streaks (
       party_id					BYTEA NOT NULL,
       active_for				INT NOT NULL,
       inactive_for				INT NOT NULL,
       is_active				BOOLEAN NOT NULL,
       reward_distribution_activity_multiplier 	TEXT NOT NULL,
       reward_vesting_activity_multiplier      	TEXT NOT NULL,
       epoch 					INT NOT NULL,
       traded_volume 				TEXT NOT NULL,
       open_volume 				TEXT NOT NULL,
       vega_time      				TIMESTAMP WITH TIME ZONE NOT NULL,
       tx_hash           			BYTEA                    NOT NULL,
       PRIMARY KEY (party_id, epoch, vega_time)
);
SELECT create_hypertable('party_activity_streaks', 'vega_time', chunk_time_interval => INTERVAL '1 day');
CREATE INDEX ON party_activity_streaks (party_id, vega_time);
CREATE INDEX ON party_activity_streaks (tx_hash);

-- +goose Down

DROP INDEX IF EXISTS party_activity_streaks_idx_tx_hash;
DROP TABLE IF EXISTS party_activity_streaks;
