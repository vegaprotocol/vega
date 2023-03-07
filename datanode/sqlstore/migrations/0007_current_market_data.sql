-- +goose Up
drop view market_data_snapshot;

create table current_market_data
(
    synthetic_time       TIMESTAMP WITH TIME ZONE NOT NULL,
    tx_hash              BYTEA                    NOT NULL,
    vega_time timestamp with time zone not null,
    seq_num    BIGINT NOT NULL,
    market bytea not null,
    mark_price HUGEINT,
    best_bid_price HUGEINT,
    best_bid_volume HUGEINT,
    best_offer_price HUGEINT,
    best_offer_volume HUGEINT,
    best_static_bid_price HUGEINT,
    best_static_bid_volume HUGEINT,
    best_static_offer_price HUGEINT,
    best_static_offer_volume HUGEINT,
    mid_price HUGEINT,
    static_mid_price HUGEINT,
    open_interest HUGEINT,
    auction_end bigint,
    auction_start bigint,
    indicative_price HUGEINT,
    indicative_volume HUGEINT,
    market_trading_mode market_trading_mode_type,
    auction_trigger auction_trigger_type,
    extension_trigger auction_trigger_type,
    target_stake HUGEINT,
    supplied_stake HUGEINT,
    price_monitoring_bounds jsonb,
    market_value_proxy text,
    liquidity_provider_fee_shares jsonb,
    market_state market_state_type,
    next_mark_to_market timestamp with time zone,
    PRIMARY KEY (market)
);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_current_market_data()
   RETURNS TRIGGER
   LANGUAGE PLPGSQL AS
$$
BEGIN
INSERT INTO current_market_data(synthetic_time,tx_hash,vega_time,seq_num,market,mark_price,best_bid_price,best_bid_volume,
                                best_offer_price,best_offer_volume,best_static_bid_price,best_static_bid_volume,
                                best_static_offer_price,best_static_offer_volume,mid_price,static_mid_price,open_interest,
                                auction_end,auction_start,indicative_price,indicative_volume,market_trading_mode,
                                auction_trigger,extension_trigger,target_stake,supplied_stake,price_monitoring_bounds,
                                market_value_proxy,liquidity_provider_fee_shares,market_state,next_mark_to_market)
                                VALUES(NEW.synthetic_time, NEW.tx_hash, NEW.vega_time, NEW.seq_num, NEW.market,
                                       NEW.mark_price, NEW.best_bid_price, NEW.best_bid_volume, NEW.best_offer_price,
                                       NEW.best_offer_volume, NEW.best_static_bid_price, NEW.best_static_bid_volume,
                                       NEW.best_static_offer_price, NEW.best_static_offer_volume, NEW.mid_price,
                                       NEW.static_mid_price, NEW.open_interest, NEW.auction_end, NEW.auction_start,
                                       NEW.indicative_price, NEW.indicative_volume, NEW.market_trading_mode,
                                       NEW.auction_trigger, NEW.extension_trigger, NEW.target_stake, NEW.supplied_stake,
                                       NEW.price_monitoring_bounds, NEW.market_value_proxy,
                                       NEW.liquidity_provider_fee_shares, NEW.market_state, NEW.next_mark_to_market)
    ON CONFLICT(market) DO UPDATE SET
    synthetic_time=EXCLUDED.synthetic_time,tx_hash=EXCLUDED.tx_hash,vega_time=EXCLUDED.vega_time,seq_num=EXCLUDED.seq_num,market=EXCLUDED.market,mark_price=EXCLUDED.mark_price,
        best_bid_price=EXCLUDED.best_bid_price,best_bid_volume=EXCLUDED.best_bid_volume,best_offer_price=EXCLUDED.best_offer_price,best_offer_volume=EXCLUDED.best_offer_volume,
        best_static_bid_price=EXCLUDED.best_static_bid_price,best_static_bid_volume=EXCLUDED.best_static_bid_volume,best_static_offer_price=EXCLUDED.best_static_offer_price,
        best_static_offer_volume=EXCLUDED.best_static_offer_volume,mid_price=EXCLUDED.mid_price,static_mid_price=EXCLUDED.static_mid_price,open_interest=EXCLUDED.open_interest,
        auction_end=EXCLUDED.auction_end,auction_start=EXCLUDED.auction_start,indicative_price=EXCLUDED.indicative_price,indicative_volume=EXCLUDED.indicative_volume,
        market_trading_mode=EXCLUDED.market_trading_mode,auction_trigger=EXCLUDED.auction_trigger,extension_trigger=EXCLUDED.extension_trigger,target_stake=EXCLUDED.target_stake,
        supplied_stake=EXCLUDED.supplied_stake,price_monitoring_bounds=EXCLUDED.price_monitoring_bounds,
        market_value_proxy=EXCLUDED.market_value_proxy,liquidity_provider_fee_shares=EXCLUDED.liquidity_provider_fee_shares,market_state=EXCLUDED.market_state,
        next_mark_to_market=EXCLUDED.next_mark_to_market;

RETURN NULL;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER update_current_market_data AFTER INSERT ON market_data FOR EACH ROW EXECUTE function update_current_market_data();

-- +goose Down
DROP TRIGGER update_current_market_data ON market_data;
DROP FUNCTION update_current_market_data;
DROP TABLE current_market_data;
