-- +goose Up
alter type auction_trigger_type add value 'AUCTION_TRIGGER_LIQUIDITY_TARGET_NOT_MET';
alter type auction_trigger_type add value 'AUCTION_TRIGGER_UNABLE_TO_DEPLOY_LP_ORDERS';


