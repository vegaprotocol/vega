-- +goose Up

-- transfer_fees table contains the fees paid on transfers. Epoch is 0 for one-off transfers.
CREATE TABLE IF NOT EXISTS transfer_fees (
      transfer_id bytea NOT NULL,
      amount HUGEINT NOT NULL,
      epoch_seq BIGINT NOT NULL,
      vega_time TIMESTAMP WITH time zone NOT NULL,
      PRIMARY KEY (vega_time, transfer_id)
);

CREATE INDEX transfer_fees_transfer_id_idx ON transfer_fees(transfer_id);

-- +goose Down
DROP INDEX transfer_fees_transfer_id_idx;
DROP TABLE IF EXISTS transfer_fees;
