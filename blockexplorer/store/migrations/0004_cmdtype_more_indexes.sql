-- +goose Up

-- This index is used when filtering on the cmd_type, while the table is
-- sorted on the block_id and the index. Not adding the block_id and the index
-- makes Postgres choose the index `tx_results_block_id_index_key` instead
-- which make the filtering on the cmd_type very slow.
CREATE INDEX tx_results_cmd_type_block_id_index ON tx_results
    USING btree (cmd_type, block_id, index);

-- +goose Down
DROP INDEX tx_results_cmd_type_block_id_index;
