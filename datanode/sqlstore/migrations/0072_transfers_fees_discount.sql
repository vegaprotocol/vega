-- +goose Up

ALTER TABLE IF EXISTS transfer_fees ADD COLUMN discount_applied HUGEINT DEFAULT (0);

-- transfer_fees_discount table contains the per party, per party available transaction fee discount.
CREATE TABLE IF NOT EXISTS transfer_fees_discount (
      party_id bytea NOT NULL,
      asset_id bytea NOT NULL,
      amount HUGEINT NOT NULL,
      epoch_seq BIGINT NOT NULL,
      vega_time TIMESTAMP WITH time zone NOT NULL,
      PRIMARY KEY (party_id, asset_id, vega_time)
);

select create_hypertable('transfer_fees_discount', 'vega_time', chunk_time_interval => INTERVAL '1 day');
CREATE INDEX transfer_fees_discount_party_id_idx ON transfer_fees_discount(party_id);
CREATE INDEX transfer_fees_discount_asset_id_idx ON transfer_fees_discount(asset_id);

-- +goose Down
ALTER TABLE transfer_fees DROP COLUMN discount_applied;
DROP INDEX transfer_fees_discount_party_id_idx;
DROP INDEX transfer_fees_discount_asset_id_idx;
DROP TABLE IF EXISTS transfer_fees_discount;
