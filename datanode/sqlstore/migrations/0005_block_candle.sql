-- +goose Up
CREATE VIEW trades_candle_block AS
SELECT market_id,  vega_time as period_start, 
       first(price, synthetic_time) AS open,
       last(price, synthetic_time) AS close,
       max(price) AS high,
       min(price) AS low,
       sum(size) AS volume,
       last(synthetic_time,
            synthetic_time) AS last_update_in_period
FROM trades
GROUP BY market_id, vega_time;

-- +goose Down
DROP VIEW trades_candle_block;