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

-- We need to make sure the next value in the rowid serial for the new tx_results table
-- continues where the old one leaves off otherwise we will break foreign key constraints
-- in the events table which we have had to drop temporarily and will restore once all the
-- data has been migrated.
do $$
    declare
        tx_results_seq_name text;
        tx_results_seq_next bigint;
    begin
        -- get the next value of the sequence for tx_results_old
        -- we will use this to reset the sequence value for the new tx_results table
        select nextval(pg_get_serial_sequence('tx_results_old', 'rowid'))
        into tx_results_seq_next;

        -- Create a new tx_results table with all the necessary fields
        CREATE TABLE tx_results (
            rowid BIGSERIAL PRIMARY KEY,
            -- The block to which this transaction belongs.
            block_id BIGINT NOT NULL REFERENCES blocks(rowid),
            -- The sequential index of the transaction within the block.
            index INTEGER NOT NULL,
            -- When this result record was logged into the sink, in UTC.
            created_at TIMESTAMPTZ NOT NULL,
            -- The hex-encoded hash of the transaction.
            tx_hash VARCHAR NOT NULL,
            -- The protobuf wire encoding of the TxResult message.
            tx_result BYTEA NOT NULL,
            submitter TEXT,
            cmd_type TEXT,
            block_height BIGINT DEFAULT 0,
            UNIQUE (block_id, index)
        );

        CREATE INDEX tx_results_tx_hash_index ON tx_results(tx_hash);
        CREATE INDEX tx_results_submitter_block_id_index_idx ON tx_results(submitter, block_id, index);
        CREATE INDEX tx_results_cmd_type_block_id_index ON tx_results
            USING btree (cmd_type, block_id, index);

        -- get the sequence name for the new tx_results serial
        select pg_get_serial_sequence('tx_results', 'rowid')
        into tx_results_seq_name;

        -- restart the sequence with the current value of the sequence for tx_results_old
        -- when nextval is called, we should get the restart value, which is the next value
        -- in the sequence for tx_results_old
        execute format('alter sequence %s restart with %s', tx_results_seq_name, tx_results_seq_next);
    end;
$$;

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

  RETURN NULL;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER add_block_height_to_tx_results
  AFTER INSERT
  ON tx_results
  FOR EACH ROW
EXECUTE PROCEDURE add_block_height_to_tx_results();

-- +goose Down

DROP TRIGGER IF EXISTS add_block_height_to_tx_results ON tx_results;

ALTER TABLE tx_results
  DROP COLUMN IF EXISTS block_height;
