-- +goose Up
CREATE INDEX tx_results_tx_hash_index ON tx_results(tx_hash);

-- +goose Down
DROP INDEX IF EXISTS tx_results_tx_hash_index
