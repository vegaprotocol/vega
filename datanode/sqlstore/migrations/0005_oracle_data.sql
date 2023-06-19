-- +goose Up
ALTER TABLE oracle_data ADD COLUMN meta_data JSONB;
ALTER TABLE oracle_data_current ADD COLUMN meta_data JSONB;

ALTER TABLE oracle_data ADD COLUMN error text;
ALTER TABLE oracle_data_current ADD COLUMN error text;


-- +goose StatementBegin

CREATE OR REPLACE FUNCTION update_current_oracle_data()
    RETURNS TRIGGER
    LANGUAGE PLPGSQL AS
$$
BEGIN
INSERT INTO oracle_data_current(signers,data,meta_data,matched_spec_ids,broadcast_at, error,tx_hash,vega_time,seq_num)
VALUES(NEW.signers,NEW.data,NEW.meta_data,NEW.matched_spec_ids,NEW.broadcast_at,NEW.error,NEW.tx_hash,NEW.vega_time,NEW.seq_num)
    ON CONFLICT(matched_spec_ids, data) DO UPDATE SET
                                               signers=EXCLUDED.signers,
                                               meta_data=EXCLUDED.meta_data,
                                               broadcast_at=EXCLUDED.broadcast_at,
                                               error=EXCLUDED.error,
                                               tx_hash=EXCLUDED.tx_hash,
                                               vega_time=EXCLUDED.vega_time,
                                               seq_num=EXCLUDED.seq_num;
RETURN NULL;
END;
$$;
-- +goose StatementEnd