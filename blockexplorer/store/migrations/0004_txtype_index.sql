
-- +goose Up
ALTER TABLE tx_results ADD column tx_type TEXT;
ALTER TABLE tx_results ADD column sender TEXT;
ALTER TABLE tx_results ADD column receiver TEXT;
CREATE INDEX tx_results_tx_type_sender_index ON tx_results(tx_type, sender);
CREATE INDEX tx_results_tx_type_receiver_index ON tx_results(tx_type, receiver);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_txresult_tx_type()
    RETURNS TRIGGER
    LANGUAGE PLPGSQL AS
$$
BEGIN
    UPDATE tx_results SET tx_type=NEW.value
    FROM events e
    WHERE e.rowid = NEW.event_ID
    AND tx_results.rowid = e.tx_id;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER update_txresult_tx_type AFTER INSERT ON attributes
    FOR EACH ROW
    WHEN (NEW.composite_key='tx.type')
    EXECUTE function update_txresult_tx_type();

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_txresult_sender()
    RETURNS TRIGGER
    LANGUAGE PLPGSQL AS
$$
BEGIN
    UPDATE tx_results SET sender=NEW.value
    FROM events e
    WHERE e.rowid = NEW.event_ID
      AND tx_results.rowid = e.tx_id;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER update_txresult_sender AFTER INSERT ON attributes
    FOR EACH ROW
    WHEN (NEW.composite_key='tx.sender')
    EXECUTE function update_txresult_sender();

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_txresult_receiver()
    RETURNS TRIGGER
    LANGUAGE PLPGSQL AS
$$
BEGIN
    UPDATE tx_results SET receiver=NEW.value
    FROM events e
    WHERE e.rowid = NEW.event_ID
      AND tx_results.rowid = e.tx_id;
    RETURN NEW;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER update_txresult_receiver AFTER INSERT ON attributes
    FOR EACH ROW
    WHEN (NEW.composite_key='tx.receiver')
    EXECUTE function update_txresult_receiver();


-- +goose Down
DROP TRIGGER update_txresult_tx_type;
DROP TRIGGER update_txresult_sender;
DROP TRIGGER update_txresult_receiver;
DROP FUNCTION IF EXISTS update_txresult_tx_type;
DROP FUNCTION IF EXISTS update_txresult_sender;
DROP FUNCTION IF EXISTS update_txresult_receiver;
DROP INDEX IF EXISTS tx_results_tx_type_sender_index;
DROP INDEX IF EXISTS tx_results_tx_type_receiver_index;
ALTER TABLE tx_results DROP COLUMN tx_type;
ALTER TABLE tx_results DROP COLUMN sender;
ALTER TABLE tx_results DROP COLUMN receiver;
