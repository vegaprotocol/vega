-- +goose Up

ALTER TYPE auction_trigger_type ADD VALUE IF NOT EXISTS 'AUCTION_TRIGGER_LONG_BLOCK';
ALTER TYPE market_trading_mode_type ADD VALUE IF NOT EXISTS 'TRADING_MODE_LONG_BLOCK_AUCTION';