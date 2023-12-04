-- +goose Up

-- make the block height nullable so tendermint can still insert and the
-- trigger takes over to set the value.
ALTER TABLE tx_results
  ADD COLUMN IF NOT EXISTS block_height BIGINT DEFAULT 0;

UPDATE tx_results
SET block_height=b.height
FROM blocks b
WHERE b.rowid = tx_results.block_id;

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

