-- +goose Up

CREATE MATERIALIZED VIEW trades_candle_1_minute_tmp
            WITH (timescaledb.continuous) AS
SELECT market_id, time_bucket('1 minute', synthetic_time) AS period_start,
       first(price, synthetic_time) AS open,
       last(price, synthetic_time) AS close,
       max(price) AS high,
       min(price) AS low,
       sum(size) AS volume,
       sum(size * price) as notional,
       last(synthetic_time,
            synthetic_time) AS last_update_in_period
FROM trades
GROUP BY market_id, period_start WITH NO DATA;

DROP MATERIALIZED VIEW IF EXISTS trades_candle_1_minute;
ALTER MATERIALIZED VIEW trades_candle_1_minute_tmp RENAME TO trades_candle_1_minute;

-- start_offset is set to a day, as data is append only this does not impact the processing time and ensures
-- that the CAGG data will be correct on recovery in the event of a transient outage ( < 1 day )
SELECT add_continuous_aggregate_policy('trades_candle_1_minute', start_offset => INTERVAL '1 day', end_offset => INTERVAL '1 minute', schedule_interval => INTERVAL '1 minute');

CREATE MATERIALIZED VIEW trades_candle_5_minutes_tmp
            WITH (timescaledb.continuous) AS
SELECT market_id, time_bucket('5 minutes', synthetic_time) AS period_start,
       first(price, synthetic_time) AS open,
       last(price, synthetic_time) AS close,
       max(price) AS high,
       min(price) AS low,
       sum(size) AS volume,
       sum(size * price) as notional,
       last(synthetic_time,
            synthetic_time) AS last_update_in_period
FROM trades
GROUP BY market_id, period_start WITH NO DATA;

DROP MATERIALIZED VIEW IF EXISTS trades_candle_5_minutes;
ALTER MATERIALIZED VIEW trades_candle_5_minutes_tmp RENAME TO trades_candle_5_minutes;

-- start_offset is set to a day, as data is append only this does not impact the processing time and ensures
-- that the CAGG data will be correct on recovery in the event of a transient outage ( < 1 day )
SELECT add_continuous_aggregate_policy('trades_candle_5_minutes', start_offset => INTERVAL '1 day', end_offset => INTERVAL '5 minutes', schedule_interval => INTERVAL '5 minutes');

CREATE MATERIALIZED VIEW trades_candle_15_minutes_tmp
            WITH (timescaledb.continuous) AS
SELECT market_id, time_bucket('15 minutes', synthetic_time) AS period_start,
       first(price, synthetic_time) AS open,
       last(price, synthetic_time) AS close,
       max(price) AS high,
       min(price) AS low,
       sum(size) AS volume,
       sum(size * price) as notional,
       last(synthetic_time,
            synthetic_time) AS last_update_in_period
FROM trades
GROUP BY market_id, period_start WITH NO DATA;

DROP MATERIALIZED VIEW IF EXISTS trades_candle_15_minutes;
ALTER MATERIALIZED VIEW trades_candle_15_minutes_tmp RENAME TO trades_candle_15_minutes;

-- start_offset is set to a day, as data is append only this does not impact the processing time and ensures
-- that the CAGG data will be correct on recovery in the event of a transient outage ( < 1 day )
SELECT add_continuous_aggregate_policy('trades_candle_15_minutes', start_offset => INTERVAL '1 day', end_offset => INTERVAL '15 minutes', schedule_interval => INTERVAL '15 minutes');

CREATE MATERIALIZED VIEW trades_candle_1_hour_tmp
            WITH (timescaledb.continuous) AS
SELECT market_id, time_bucket('1 hour', synthetic_time) AS period_start,
       first(price, synthetic_time) AS open,
       last(price, synthetic_time) AS close,
       max(price) AS high,
       min(price) AS low,
       sum(size) AS volume,
       sum(size * price) as notional,
       last(synthetic_time,
            synthetic_time) AS last_update_in_period
FROM trades
GROUP BY market_id, period_start WITH NO DATA;

DROP MATERIALIZED VIEW IF EXISTS trades_candle_1_hour;
ALTER MATERIALIZED VIEW trades_candle_1_hour_tmp RENAME TO trades_candle_1_hour;

-- start_offset is set to a day, as data is append only this does not impact the processing time and ensures
-- that the CAGG data will be correct on recovery in the event of a transient outage ( < 1 day )
SELECT add_continuous_aggregate_policy('trades_candle_1_hour', start_offset => INTERVAL '1 day', end_offset => INTERVAL '1 hour', schedule_interval => INTERVAL '1 hour');

CREATE MATERIALIZED VIEW trades_candle_6_hours_tmp
            WITH (timescaledb.continuous) AS
SELECT market_id, time_bucket('6 hours', synthetic_time) AS period_start,
       first(price, synthetic_time) AS open,
       last(price, synthetic_time) AS close,
       max(price) AS high,
       min(price) AS low,
       sum(size) AS volume,
       sum(size * price) as notional,
       last(synthetic_time,
            synthetic_time) AS last_update_in_period
FROM trades
GROUP BY market_id, period_start WITH NO DATA;

DROP MATERIALIZED VIEW IF EXISTS trades_candle_6_hours;
ALTER MATERIALIZED VIEW trades_candle_6_hours_tmp RENAME TO trades_candle_6_hours;

-- start_offset is set to a day, as data is append only this does not impact the processing time and ensures
-- that the CAGG data will be correct on recovery in the event of a transient outage ( < 1 day )
SELECT add_continuous_aggregate_policy('trades_candle_6_hours', start_offset => INTERVAL '1 day', end_offset => INTERVAL '6 hours', schedule_interval => INTERVAL '6 hours');

CREATE MATERIALIZED VIEW trades_candle_1_day_tmp
            WITH (timescaledb.continuous) AS
SELECT market_id, time_bucket('1 day', synthetic_time) AS period_start,
       first(price, synthetic_time) AS open,
       last(price, synthetic_time) AS close,
       max(price) AS high,
       min(price) AS low,
       sum(size) AS volume,
       sum(size * price) as notional,
       last(synthetic_time,
            synthetic_time) AS last_update_in_period
FROM trades
GROUP BY market_id, period_start WITH NO DATA;

DROP MATERIALIZED VIEW IF EXISTS trades_candle_1_day;
ALTER MATERIALIZED VIEW trades_candle_1_day_tmp RENAME TO trades_candle_1_day;

SELECT add_continuous_aggregate_policy('trades_candle_1_day', start_offset => INTERVAL '3 days', end_offset => INTERVAL '1 day', schedule_interval => INTERVAL '1 day');

CREATE OR REPLACE VIEW trades_candle_block AS
SELECT market_id,  vega_time as period_start,
       first(price, synthetic_time) AS open,
       last(price, synthetic_time) AS close,
       max(price) AS high,
       min(price) AS low,
       sum(size) AS volume,
       sum(size * price) as notional,
       last(synthetic_time,
            synthetic_time) AS last_update_in_period
FROM trades
GROUP BY market_id, vega_time;

-- +goose Down

DROP VIEW IF EXISTS trades_candle_block;
