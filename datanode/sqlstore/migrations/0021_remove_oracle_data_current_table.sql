-- +goose Up

drop trigger if exists update_current_oracle_data on oracle_data;
drop function if exists update_current_oracle_data;
DROP TABLE IF EXISTS oracle_data_current;

-- +goose Down


create table if not exists oracle_data_current (
                                                   signers bytea[],
                                                   data jsonb not null,
                                                   matched_spec_ids bytea[],
                                                   broadcast_at timestamp with time zone not null,
                                                   tx_hash  bytea not null,
                                                   vega_time timestamp with time zone not null,
                                                   seq_num  BIGINT NOT NULL,
                                                   PRIMARY KEY(matched_spec_ids, data)
    );

-- +goose StatementBegin

CREATE OR REPLACE FUNCTION update_current_oracle_data()
    RETURNS TRIGGER
    LANGUAGE PLPGSQL AS
$$
BEGIN
INSERT INTO oracle_data_current(signers,data,matched_spec_ids,broadcast_at,tx_hash,vega_time,seq_num)
VALUES(NEW.signers,NEW.data,NEW.matched_spec_ids,NEW.broadcast_at,NEW.tx_hash,NEW.vega_time,NEW.seq_num)
    ON CONFLICT(matched_spec_ids, data) DO UPDATE SET
    signers=EXCLUDED.signers,
                                               broadcast_at=EXCLUDED.broadcast_at,
                                               tx_hash=EXCLUDED.tx_hash,
                                               vega_time=EXCLUDED.vega_time,
                                               seq_num=EXCLUDED.seq_num;
RETURN NULL;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER update_current_oracle_data AFTER INSERT ON oracle_data FOR EACH ROW EXECUTE function update_current_oracle_data();