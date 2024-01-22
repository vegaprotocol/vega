-- +goose Up

-- make the block height nullable so tendermint can still insert and the
-- trigger takes over to set the value.
ALTER TABLE tx_results
  ADD COLUMN IF NOT EXISTS block_height BIGINT DEFAULT 0;

-- First drop any foreign key constraints that depend on the tx_results table
-- This will be restored after all the data has been migrated to the new tx_results table
ALTER TABLE events DROP constraint events_tx_id_fkey;

-- Rename the tx_results table to tx_results_old
ALTER TABLE tx_results RENAME TO tx_results_old;
ALTER INDEX tx_results_tx_hash_index RENAME TO tx_results_old_tx_hash_index;
ALTER INDEX tx_results_submitter_block_id_index_idx RENAME TO tx_results_old_submitter_block_id_index_idx;
ALTER INDEX tx_results_cmd_type_block_id_index RENAME TO tx_results_old_cmd_type_block_id_index;

-- +goose Down

ALTER INDEX tx_results_old_cmd_type_block_id_index RENAME TO tx_results_cmd_type_block_id_index;
ALTER INDEX tx_results_old_submitter_block_id_index_idx RENAME TO tx_results_submitter_block_id_index_idx;
ALTER INDEX tx_results_old_tx_hash_index RENAME TO tx_results_tx_hash_index;
ALTER TABLE tx_results_old RENAME TO tx_results;

ALTER TABLE events ADD constraint events_tx_id_fkey FOREIGN KEY (tx_id) REFERENCES tx_results(rowid);

ALTER TABLE tx_results
    DROP COLUMN IF EXISTS block_height;

-- Recreate views, functions and triggers associated with the original tx_results table
CREATE OR REPLACE VIEW tx_events AS
SELECT height, index, chain_id, type, key, composite_key, value, tx_results.created_at
FROM blocks JOIN tx_results ON (blocks.rowid = tx_results.block_id)
            JOIN event_attributes ON (tx_results.rowid = event_attributes.tx_id)
WHERE event_attributes.tx_id IS NOT NULL;


-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_txresult_cmd_type()
    RETURNS TRIGGER
    LANGUAGE PLPGSQL AS
$$
BEGIN
    UPDATE tx_results SET cmd_type=NEW.value
    FROM events e
    WHERE e.rowid = NEW.event_ID
      AND tx_results.rowid = e.tx_id;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd


-- +goose StatementBegin
CREATE OR REPLACE FUNCTION add_block_height_to_tx_results()
    RETURNS TRIGGER
    LANGUAGE plpgsql AS
$$
BEGIN
    UPDATE tx_results
    SET block_height=b.height
    FROM blocks b
    WHERE b.rowid = NEW.block_id
      AND tx_results.rowid = NEW.rowid;

    RETURN NEW;
END;
$$;
-- +goose StatementEnd
