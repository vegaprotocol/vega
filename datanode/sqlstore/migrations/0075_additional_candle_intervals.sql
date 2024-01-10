-- +goose Up

CREATE MATERIALIZED VIEW trades_candle_30_minutes
            WITH (timescaledb.continuous) AS
SELECT market_id, time_bucket('30 minute', synthetic_time) AS period_start,
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
SELECT add_continuous_aggregate_policy('trades_candle_30_minutes', start_offset => INTERVAL '1 day', end_offset => INTERVAL '30 minutes', schedule_interval => INTERVAL '30 minutes');

CREATE MATERIALIZED VIEW trades_candle_4_hours
            WITH (timescaledb.continuous) AS
SELECT market_id, time_bucket('4 hours', synthetic_time) AS period_start,
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
SELECT add_continuous_aggregate_policy('trades_candle_4_hours', start_offset => INTERVAL '1 day', end_offset => INTERVAL '4 hours', schedule_interval => INTERVAL '4 hours');

CREATE MATERIALIZED VIEW trades_candle_8_hours
            WITH (timescaledb.continuous) AS
SELECT market_id, time_bucket('8 hours', synthetic_time) AS period_start,
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
SELECT add_continuous_aggregate_policy('trades_candle_8_hours', start_offset => INTERVAL '1 day', end_offset => INTERVAL '8 hours', schedule_interval => INTERVAL '8 hours');

CREATE MATERIALIZED VIEW trades_candle_12_hours
            WITH (timescaledb.continuous) AS
SELECT market_id, time_bucket('12 hours', synthetic_time) AS period_start,
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
SELECT add_continuous_aggregate_policy('trades_candle_12_hours', start_offset => INTERVAL '2 days', end_offset => INTERVAL '12 hours', schedule_interval => INTERVAL '12 hours');

CREATE MATERIALIZED VIEW trades_candle_7_days
            WITH (timescaledb.continuous) AS
SELECT market_id, time_bucket('7 days', synthetic_time) AS period_start,
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
SELECT add_continuous_aggregate_policy('trades_candle_7_days', start_offset => INTERVAL '21 days', end_offset => INTERVAL '7 days', schedule_interval => INTERVAL '7 days');



-- +goose Down
SELECT remove_continuous_aggregate_policy('trades_candle_7_days', true);

DROP MATERIALIZED VIEW trades_candle_7_days;

SELECT remove_continuous_aggregate_policy('trades_candle_12_hours', true);

DROP MATERIALIZED VIEW trades_candle_12_hours;

SELECT remove_continuous_aggregate_policy('trades_candle_8_hours', true);

DROP MATERIALIZED VIEW trades_candle_8_hours;

SELECT remove_continuous_aggregate_policy('trades_candle_4_hours', true);

DROP MATERIALIZED VIEW trades_candle_4_hours;

SELECT remove_continuous_aggregate_policy('trades_candle_30_minutes', true);

DROP MATERIALIZED VIEW trades_candle_30_minutes;
