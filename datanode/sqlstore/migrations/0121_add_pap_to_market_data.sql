-- +goose Up
alter table market_data add column if not exists active_protocol_automated_purchase JSONB;
alter table current_market_data add column if not exists active_protocol_automated_purchase JSONB;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_current_market_data()
    RETURNS TRIGGER
    LANGUAGE PLPGSQL AS
$$
BEGIN
INSERT INTO current_market_data(synthetic_time,tx_hash,vega_time,seq_num,market,mark_price,mark_price_type,best_bid_price,best_bid_volume,
                                best_offer_price,best_offer_volume,best_static_bid_price,best_static_bid_volume,
                                best_static_offer_price,best_static_offer_volume,mid_price,static_mid_price,open_interest,
                                auction_end,auction_start,indicative_price,indicative_volume,market_trading_mode,
                                auction_trigger,extension_trigger,target_stake,supplied_stake,price_monitoring_bounds,
                                market_value_proxy,liquidity_provider_fee_shares,market_state,next_mark_to_market, market_growth, last_traded_price, product_data, liquidity_provider_sla, next_network_closeout, mark_price_state, active_protocol_automated_purchase)
VALUES(NEW.synthetic_time, NEW.tx_hash, NEW.vega_time, NEW.seq_num, NEW.market,
       NEW.mark_price, NEW.mark_price_type, NEW.best_bid_price, NEW.best_bid_volume, NEW.best_offer_price,
       NEW.best_offer_volume, NEW.best_static_bid_price, NEW.best_static_bid_volume,
       NEW.best_static_offer_price, NEW.best_static_offer_volume, NEW.mid_price,
       NEW.static_mid_price, NEW.open_interest, NEW.auction_end, NEW.auction_start,
       NEW.indicative_price, NEW.indicative_volume, NEW.market_trading_mode,
       NEW.auction_trigger, NEW.extension_trigger, NEW.target_stake, NEW.supplied_stake,
       NEW.price_monitoring_bounds, NEW.market_value_proxy,
       NEW.liquidity_provider_fee_shares, NEW.market_state, NEW.next_mark_to_market, NEW.market_growth, NEW.last_traded_price, NEW.product_data, NEW.liquidity_provider_sla, NEW.next_network_closeout, NEW.mark_price_state, NEW.active_protocol_automated_purchase)
    ON CONFLICT(market) DO UPDATE SET
    synthetic_time=EXCLUDED.synthetic_time,tx_hash=EXCLUDED.tx_hash,vega_time=EXCLUDED.vega_time,seq_num=EXCLUDED.seq_num,market=EXCLUDED.market,mark_price=EXCLUDED.mark_price,
                               mark_price_type=EXCLUDED.mark_price_type,
                               best_bid_price=EXCLUDED.best_bid_price,best_bid_volume=EXCLUDED.best_bid_volume,best_offer_price=EXCLUDED.best_offer_price,best_offer_volume=EXCLUDED.best_offer_volume,
                               best_static_bid_price=EXCLUDED.best_static_bid_price,best_static_bid_volume=EXCLUDED.best_static_bid_volume,best_static_offer_price=EXCLUDED.best_static_offer_price,
                               best_static_offer_volume=EXCLUDED.best_static_offer_volume,mid_price=EXCLUDED.mid_price,static_mid_price=EXCLUDED.static_mid_price,open_interest=EXCLUDED.open_interest,
                               auction_end=EXCLUDED.auction_end,auction_start=EXCLUDED.auction_start,indicative_price=EXCLUDED.indicative_price,indicative_volume=EXCLUDED.indicative_volume,
                               market_trading_mode=EXCLUDED.market_trading_mode,auction_trigger=EXCLUDED.auction_trigger,extension_trigger=EXCLUDED.extension_trigger,target_stake=EXCLUDED.target_stake,
                               supplied_stake=EXCLUDED.supplied_stake,price_monitoring_bounds=EXCLUDED.price_monitoring_bounds,
                               market_value_proxy=EXCLUDED.market_value_proxy,liquidity_provider_fee_shares=EXCLUDED.liquidity_provider_fee_shares,market_state=EXCLUDED.market_state,
                               next_mark_to_market=EXCLUDED.next_mark_to_market,market_growth=EXCLUDED.market_growth,last_traded_price=EXCLUDED.last_traded_price,
                               product_data=EXCLUDED.product_data,liquidity_provider_sla=EXCLUDED.liquidity_provider_sla,next_network_closeout=EXCLUDED.next_network_closeout,
                               mark_price_state=EXCLUDED.mark_price_state, active_protocol_automated_purchase=EXCLUDED.active_protocol_automated_purchase;

RETURN NULL;
END;
$$;
-- +goose StatementEnd

-- +goose Down
alter table current_market_data drop column if exists active_protocol_automated_purchase;
alter table market_data drop column if exists active_protocol_automated_purchase;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_current_market_data()
    RETURNS TRIGGER
    LANGUAGE PLPGSQL AS
$$
BEGIN
INSERT INTO current_market_data(synthetic_time,tx_hash,vega_time,seq_num,market,mark_price,mark_price_type,best_bid_price,best_bid_volume,
                                best_offer_price,best_offer_volume,best_static_bid_price,best_static_bid_volume,
                                best_static_offer_price,best_static_offer_volume,mid_price,static_mid_price,open_interest,
                                auction_end,auction_start,indicative_price,indicative_volume,market_trading_mode,
                                auction_trigger,extension_trigger,target_stake,supplied_stake,price_monitoring_bounds,
                                market_value_proxy,liquidity_provider_fee_shares,market_state,next_mark_to_market, market_growth, last_traded_price, product_data, liquidity_provider_sla, next_network_closeout, mark_price_state)
VALUES(NEW.synthetic_time, NEW.tx_hash, NEW.vega_time, NEW.seq_num, NEW.market,
       NEW.mark_price, NEW.mark_price_type, NEW.best_bid_price, NEW.best_bid_volume, NEW.best_offer_price,
       NEW.best_offer_volume, NEW.best_static_bid_price, NEW.best_static_bid_volume,
       NEW.best_static_offer_price, NEW.best_static_offer_volume, NEW.mid_price,
       NEW.static_mid_price, NEW.open_interest, NEW.auction_end, NEW.auction_start,
       NEW.indicative_price, NEW.indicative_volume, NEW.market_trading_mode,
       NEW.auction_trigger, NEW.extension_trigger, NEW.target_stake, NEW.supplied_stake,
       NEW.price_monitoring_bounds, NEW.market_value_proxy,
       NEW.liquidity_provider_fee_shares, NEW.market_state, NEW.next_mark_to_market, NEW.market_growth, NEW.last_traded_price, NEW.product_data, NEW.liquidity_provider_sla, NEW.next_network_closeout, NEW.mark_price_state)
    ON CONFLICT(market) DO UPDATE SET
    synthetic_time=EXCLUDED.synthetic_time,tx_hash=EXCLUDED.tx_hash,vega_time=EXCLUDED.vega_time,seq_num=EXCLUDED.seq_num,market=EXCLUDED.market,mark_price=EXCLUDED.mark_price,
                               mark_price_type=EXCLUDED.mark_price_type,
                               best_bid_price=EXCLUDED.best_bid_price,best_bid_volume=EXCLUDED.best_bid_volume,best_offer_price=EXCLUDED.best_offer_price,best_offer_volume=EXCLUDED.best_offer_volume,
                               best_static_bid_price=EXCLUDED.best_static_bid_price,best_static_bid_volume=EXCLUDED.best_static_bid_volume,best_static_offer_price=EXCLUDED.best_static_offer_price,
                               best_static_offer_volume=EXCLUDED.best_static_offer_volume,mid_price=EXCLUDED.mid_price,static_mid_price=EXCLUDED.static_mid_price,open_interest=EXCLUDED.open_interest,
                               auction_end=EXCLUDED.auction_end,auction_start=EXCLUDED.auction_start,indicative_price=EXCLUDED.indicative_price,indicative_volume=EXCLUDED.indicative_volume,
                               market_trading_mode=EXCLUDED.market_trading_mode,auction_trigger=EXCLUDED.auction_trigger,extension_trigger=EXCLUDED.extension_trigger,target_stake=EXCLUDED.target_stake,
                               supplied_stake=EXCLUDED.supplied_stake,price_monitoring_bounds=EXCLUDED.price_monitoring_bounds,
                               market_value_proxy=EXCLUDED.market_value_proxy,liquidity_provider_fee_shares=EXCLUDED.liquidity_provider_fee_shares,market_state=EXCLUDED.market_state,
                               next_mark_to_market=EXCLUDED.next_mark_to_market,market_growth=EXCLUDED.market_growth,last_traded_price=EXCLUDED.last_traded_price,
                               product_data=EXCLUDED.product_data,liquidity_provider_sla=EXCLUDED.liquidity_provider_sla,next_network_closeout=EXCLUDED.next_network_closeout,
                               mark_price_state=EXCLUDED.mark_price_state;
RETURN NULL;
END;
$$;
-- +goose StatementEnd
