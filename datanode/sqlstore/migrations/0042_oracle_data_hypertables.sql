-- +goose Up
-- So because I think of some obscure bug in timescaledb, we can't simply 
-- create_hypertable ('oracle_data', ...), because in the past it had a previous
-- column, 'matched_spec_ids'. When you drop a column in postgres, I don't think
-- it actually goes away, it just gets flagged as 'removed' and the CSV import
-- doesn't work, complaining about 
-- - attribute 6 of type _timescaledb_internal._hyper_184_47_chunk has wrong type
-- - 'Table has type bigint, but query expects timestamp with time zone.'
-- So we have to create a new table, copy the data in, drop the old table, and rename.
   CREATE TABLE oracle_data_temp (LIKE oracle_data INCLUDING ALL);

   INSERT INTO oracle_data_temp
   SELECT *
     FROM oracle_data;

     DROP TABLE oracle_data;

    ALTER TABLE oracle_data_temp
RENAME TO oracle_data;

   SELECT create_hypertable ('oracle_data', 'vega_time', chunk_time_interval => INTERVAL '1 day', migrate_data => TRUE);

-- oracle_data_oracle_specs is fine though we don't have to faff around with it because we never dropped any columns
   SELECT create_hypertable ('oracle_data_oracle_specs', 'vega_time', chunk_time_interval => INTERVAL '1 day', migrate_data => TRUE);

-- +goose Down
-- to de-hypertable ourselves we have to make a new table and copy stuff in
-- It would be nice to create the new table like this but it confuses timescaledb
   CREATE TABLE oracle_data_temp (LIKE oracle_data INCLUDING ALL);

   INSERT INTO oracle_data_temp
   SELECT *
     FROM oracle_data;

     DROP TABLE oracle_data;

    ALTER TABLE oracle_data_temp
RENAME TO oracle_data;

-- same for the join table to specs
   CREATE TABLE oracle_data_oracle_specs_temp (LIKE oracle_data_oracle_specs INCLUDING ALL);

   INSERT INTO oracle_data_oracle_specs_temp
   SELECT *
     FROM oracle_data_oracle_specs;

     DROP TABLE oracle_data_oracle_specs;

    ALTER TABLE oracle_data_oracle_specs_temp
RENAME TO oracle_data_oracle_specs;