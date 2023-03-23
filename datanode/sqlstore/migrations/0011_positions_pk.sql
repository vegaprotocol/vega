-- +goose Up
ALTER TABLE positions DROP CONSTRAINT positions_pkey;
ALTER TABLE positions ADD PRIMARY KEY (vega_time, party_id, market_id);
CREATE INDEX ON positions(party_id, market_id, vega_time);

-- +goose Down
DROP INDEX IF EXISTS positions_party_id_market_id_vega_time_idx;
ALTER TABLE positions DROP CONSTRAINT positions_pkey;
ALTER TABLE positions ADD PRIMARY KEY (party_id, market_id, vega_time);

