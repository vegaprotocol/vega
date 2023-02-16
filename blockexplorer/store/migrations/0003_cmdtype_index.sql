
-- +goose Up
ALTER TABLE tx_results ADD column cmd_type TEXT;
CREATE INDEX tx_results_cmd_type_index ON tx_results(cmd_type, submitter);

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

CREATE TRIGGER update_txresult_cmd_type AFTER INSERT ON attributes
    FOR EACH ROW
    WHEN (NEW.composite_key='command.type')
    EXECUTE function update_txresult_cmd_type();


-- +goose Down
DROP TRIGGER update_txresult_cmd_type;
DROP FUNCTION IF EXISTS update_txresult_cmd_type;
DROP INDEX IF EXISTS tx_results_cmd_type_index;
ALTER TABLE tx_results DROP COLUMN cmd_type;