-- +goose Up
ALTER TABLE tx_results ADD column submitter TEXT;

CREATE INDEX tx_results_tx_hash_index ON tx_results(tx_hash);
CREATE INDEX tx_results_submitter_block_id_index_idx ON tx_results(submitter, block_id, index);
CREATE INDEX attributes_value_index ON attributes(value, composite_key);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_txresult_submitter()
    RETURNS TRIGGER
    LANGUAGE PLPGSQL AS
$$
BEGIN
    UPDATE tx_results SET submitter=NEW.value
    FROM events e
    WHERE e.rowid = NEW.event_ID
    AND tx_results.rowid = e.tx_id;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER update_txresult_submitter AFTER INSERT ON attributes
    FOR EACH ROW
    WHEN (NEW.composite_key='tx.submitter')
    EXECUTE function update_txresult_submitter();


-- +goose Down
DROP TRIGGER update_txresult_submitter;
DROP FUNCTION IF EXISTS update_txresult_submitter;

DROP INDEX IF EXISTS attributes_value_index;
DROP INDEX IF EXISTS tx_results_submitter_block_id_index_idx;
DROP INDEX IF EXISTS tx_results_tx_hash_index;

ALTER TABLE tx_results DROP COLUMN submitter;