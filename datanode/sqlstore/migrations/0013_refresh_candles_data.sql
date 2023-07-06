-- +goose Up
-- +goose NO TRANSACTION

CALL refresh_continuous_aggregate('trades_candle_1_minute', null, null);
CALL refresh_continuous_aggregate('trades_candle_5_minutes', null, null);
CALL refresh_continuous_aggregate('trades_candle_15_minutes', null, null);
CALL refresh_continuous_aggregate('trades_candle_1_hour', null, null);
CALL refresh_continuous_aggregate('trades_candle_6_hours', null, null);
CALL refresh_continuous_aggregate('trades_candle_1_day', null, null);