-- An earlier migration (0004_oracle_specs) changed the schema the oracle_specs so that instead of having two columns
-- for signers and filters, we have a single column 'data', which is a JSONB object; in order to support more diverse
-- sorts of data (e.g. from ethereum oracles)
--
-- Unforunately, we neglected to preseve the existing data in those columns, so they were left with NULL values in data.
-- A recent API change meant that this has started to cause issues on the front end.
--
-- Thankfully this information is actually redundantly stored inside the 'tradable_instument' field of the market table,
-- so we are able to recreate it; which is what this migration does.
--
-- We found a couple of cases where there the data was missing in the market table, so for those we simply delete the
-- rows; they are not critical and new ones will be written when the oracle changes state.
-- Fish out the relevant stuff from our markets
-- +goose Up
     WITH instruments AS (
             SELECT tradable_instrument -> 'instrument' AS instrument
               FROM markets_current
          ),
          futures AS (
             SELECT instrument -> 'future' AS future
               FROM instruments
              WHERE instrument ? 'future'
          ),
          futures_settlement AS (
             SELECT future -> 'dataSourceSpecForSettlementData' AS spec
               FROM futures
              WHERE future ? 'dataSourceSpecForSettlementData'
          ),
          futures_termination AS (
             SELECT future -> 'dataSourceSpecForTradingTermination' AS spec
               FROM futures
              WHERE future ? 'dataSourceSpecForTradingTermination'
          ),
          perpetuals AS (
             SELECT instrument -> 'perpetual' AS perpetual
               FROM instruments
              WHERE instrument ? 'perpetual'
          ),
          perpetuals_settlement AS (
             SELECT perpetual -> 'dataSourceSpecForSettlementData' AS spec
               FROM perpetuals
              WHERE perpetual ? 'dataSourceSpecForSettlementData'
          ),
          perpetuals_schedule AS (
             SELECT perpetual -> 'dataSourceSpecForSettlementSchedule' AS spec
               FROM perpetuals
              WHERE perpetual ? 'dataSourceSpecForSettlementSchedule'
          ),
          all_specs AS (
             SELECT *
               FROM futures_settlement
          UNION ALL
             SELECT *
               FROM futures_termination
          UNION ALL
             SELECT *
               FROM perpetuals_settlement
          UNION ALL
             SELECT *
               FROM perpetuals_schedule
          ),
          nice_specs AS (
             SELECT DECODE(spec ->> 'id', 'hex') AS id,
                    spec -> 'data'               AS DATA
               FROM all_specs
          ),
          unique_specs AS (
             SELECT DISTINCT ON (id) *
               FROM nice_specs
          ),
          changes AS (
             SELECT              os.id,
                    os.vega_time,
                    os.data      AS old_data,
                    us.data      AS new_data
               FROM oracle_specs os
          LEFT JOIN unique_specs us ON os.id = us.id
              WHERE os.data IS NULL
                AND us.data IS NOT NULL
           ORDER BY os.id,
                    os.vega_time
          )
   UPDATE oracle_specs os
      SET DATA = changes.new_data
     FROM changes
    WHERE os.id = changes.id
      AND os.vega_time = changes.vega_time;

-- Finally remove any straggelers 
   DELETE FROM oracle_specs
    WHERE DATA IS NULL;

-- And lets make sure it can't happen again
    ALTER TABLE oracle_specs
    ALTER COLUMN DATA
      SET NOT NULL;

-- +goose Down
    ALTER TABLE oracle_specs
    ALTER COLUMN DATA
     DROP NOT NULL;