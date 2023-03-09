-- +goose Up
alter type auction_trigger_type add value 'AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET';
alter type auction_trigger_type add value 'AUCTION_TRIGGER_UNABLE_TO_DEPLOY_LP_ORDERS';

-- +goose Down
ALTER TYPE auction_trigger_type RENAME TO auction_trigger_type_old;
UPDATE market_data SET auction_trigger = 'AUCTION_TRIGGER_LIQUIDITY' WHERE auction_trigger in ('AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET','AUCTION_TRIGGER_UNABLE_TO_DEPLOY_LP_ORDERS');
UPDATE market_data SET extension_trigger = 'AUCTION_TRIGGER_LIQUIDITY' WHERE extension_trigger in ('AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET','AUCTION_TRIGGER_UNABLE_TO_DEPLOY_LP_ORDERS');
UPDATE current_market_data SET auction_trigger = 'AUCTION_TRIGGER_LIQUIDITY' WHERE auction_trigger in ('AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET','AUCTION_TRIGGER_UNABLE_TO_DEPLOY_LP_ORDERS');
UPDATE current_market_data SET extension_trigger = 'AUCTION_TRIGGER_LIQUIDITY' WHERE extension_trigger in ('AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET','AUCTION_TRIGGER_UNABLE_TO_DEPLOY_LP_ORDERS');
create type auction_trigger_type as enum('AUCTION_TRIGGER_UNSPECIFIED', 'AUCTION_TRIGGER_BATCH', 'AUCTION_TRIGGER_OPENING', 'AUCTION_TRIGGER_PRICE', 'AUCTION_TRIGGER_LIQUIDITY');
ALTER TABLE market_data ALTER COLUMN auction_trigger TYPE auction_trigger_type USING auction_trigger::text::auction_trigger_type;
ALTER TABLE market_data ALTER COLUMN extension_trigger TYPE auction_trigger_type USING extension_trigger::text::auction_trigger_type;
ALTER TABLE current_market_data ALTER COLUMN auction_trigger TYPE auction_trigger_type USING auction_trigger::text::auction_trigger_type;
ALTER TABLE current_market_data ALTER COLUMN extension_trigger TYPE auction_trigger_type USING extension_trigger::text::auction_trigger_type;
DROP TYPE auction_trigger_type_old;