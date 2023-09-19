-- +goose Up
-- Make our new join table to link oracle data to oracle specs
CREATE TABLE IF NOT EXISTS
    oracle_data_oracle_specs (
        vega_time TIMESTAMP WITH TIME ZONE NOT NULL,
        seq_num BIGINT NOT NULL,
        spec_id bytea NOT NULL,
        PRIMARY KEY (vega_time, seq_num, spec_id)
    );

CREATE INDEX IF NOT EXISTS idx_oracle_data_oracle_specs_spec_to_data ON oracle_data_oracle_specs (spec_id);

-- Populate it with the contents of the 'matched_spec_ids' bytea array column
INSERT INTO
    oracle_data_oracle_specs
SELECT
    vega_time,
    seq_num,
    UNNEST(matched_spec_ids)
FROM
    oracle_data;

-- Remove old column
ALTER TABLE oracle_data
DROP COLUMN matched_spec_ids;

-- +goose Down
-- Add back the old column
ALTER TABLE oracle_data
ADD COLUMN matched_spec_ids bytea[];

-- Populate it with the contents of the oracle_data_oracle_specs table
WITH
    stuff AS (
        SELECT
            ARRAY_AGG(spec_id) AS sids,
            vega_time AS vt,
            seq_num AS sn
        FROM
            oracle_data_oracle_specs
        GROUP BY
            vega_time,
            seq_num
    )
UPDATE oracle_data
SET
    matched_spec_ids = sids
FROM
    stuff
WHERE
    vega_time = vt
    AND seq_num = sn;

-- Remove the join table
DROP TABLE oracle_data_oracle_specs;