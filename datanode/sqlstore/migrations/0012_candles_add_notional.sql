-- +goose Up

SELECT remove_continuous_aggregate_policy('trades_candle_1_minute');
DROP MATERIALIZED VIEW IF EXISTS trades_candle_1_minute;

CREATE MATERIALIZED VIEW trades_candle_1_minute
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

-- start_offset is set to a day, as data is append only this does not impact the processing time and ensures
-- that the CAGG data will be correct on recovery in the event of a transient outage ( < 1 day )
SELECT refresh_continuous_aggregate('trades_candle_1_minute', null, null);
SELECT add_continuous_aggregate_policy('trades_candle_1_minute', start_offset => INTERVAL '1 day', end_offset => INTERVAL '1 minute', schedule_interval => INTERVAL '1 minute');

SELECT remove_continuous_aggregate_policy('trades_candle_5_minutes');
DROP MATERIALIZED VIEW IF EXISTS trades_candle_5_minutes;

CREATE MATERIALIZED VIEW trades_candle_5_minutes
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

-- start_offset is set to a day, as data is append only this does not impact the processing time and ensures
-- that the CAGG data will be correct on recovery in the event of a transient outage ( < 1 day )
SELECT refresh_continuous_aggregate('trades_candle_5_minutes', null, null);
SELECT add_continuous_aggregate_policy('trades_candle_5_minutes', start_offset => INTERVAL '1 day', end_offset => INTERVAL '5 minutes', schedule_interval => INTERVAL '5 minutes');

SELECT remove_continuous_aggregate_policy('trades_candle_15_minutes');
DROP MATERIALIZED VIEW IF EXISTS trades_candle_15_minutes;

CREATE MATERIALIZED VIEW trades_candle_15_minutes
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

-- start_offset is set to a day, as data is append only this does not impact the processing time and ensures
-- that the CAGG data will be correct on recovery in the event of a transient outage ( < 1 day )
SELECT refresh_continuous_aggregate('trades_candle_15_minutes', null, null);
SELECT add_continuous_aggregate_policy('trades_candle_15_minutes', start_offset => INTERVAL '1 day', end_offset => INTERVAL '15 minutes', schedule_interval => INTERVAL '15 minutes');

SELECT remove_continuous_aggregate_policy('trades_candle_1_hour');
DROP MATERIALIZED VIEW IF EXISTS trades_candle_1_hour;

CREATE MATERIALIZED VIEW trades_candle_1_hour
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

-- start_offset is set to a day, as data is append only this does not impact the processing time and ensures
-- that the CAGG data will be correct on recovery in the event of a transient outage ( < 1 day )
SELECT refresh_continuous_aggregate('trades_candle_1_hour', null, null);
SELECT add_continuous_aggregate_policy('trades_candle_1_hour', start_offset => INTERVAL '1 day', end_offset => INTERVAL '1 hour', schedule_interval => INTERVAL '1 hour');

SELECT remove_continuous_aggregate_policy('trades_candle_6_hours');
DROP MATERIALIZED VIEW IF EXISTS trades_candle_6_hours;

CREATE MATERIALIZED VIEW trades_candle_6_hours
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

-- start_offset is set to a day, as data is append only this does not impact the processing time and ensures
-- that the CAGG data will be correct on recovery in the event of a transient outage ( < 1 day )
SELECT refresh_continuous_aggregate('trades_candle_6_hours', null, null);
SELECT add_continuous_aggregate_policy('trades_candle_6_hours', start_offset => INTERVAL '1 day', end_offset => INTERVAL '6 hours', schedule_interval => INTERVAL '6 hours');

SELECT remove_continuous_aggregate_policy('trades_candle_1_day');
DROP MATERIALIZED VIEW IF EXISTS trades_candle_1_day;

CREATE MATERIALIZED VIEW trades_candle_1_day
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

SELECT refresh_continuous_aggregate('trades_candle_1_day', null, null);
SELECT add_continuous_aggregate_policy('trades_candle_1_day', start_offset => INTERVAL '3 days', end_offset => INTERVAL '1 day', schedule_interval => INTERVAL '1 day');

CREATE OR REPLACE VIEW trades_candle_block AS
SELECT market_id,  vega_time as period_start,
       first(price, synthetic_time) AS open,
       last(price, synthetic_time) AS close,
       max(price) AS high,
       min(price) AS low,
       sum(size) AS volume,
       last(synthetic_time, synthetic_time) AS last_update_in_period,
       sum(size * price) as notional

FROM trades
GROUP BY market_id, vega_time;

-- +goose Down

DROP VIEW IF EXISTS trades_candle_block;
