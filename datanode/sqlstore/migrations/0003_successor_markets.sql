-- +goose Up
ALTER TABLE markets
    ADD IF NOT EXISTS parent_market_id BYTEA NULL,
    ADD IF NOT EXISTS insurance_pool_fraction NUMERIC NULL
;

ALTER TABLE markets_current
    ADD IF NOT EXISTS parent_market_id BYTEA NULL,
    ADD IF NOT EXISTS insurance_pool_fraction NUMERIC NULL
;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_current_markets()
    RETURNS TRIGGER
    LANGUAGE PLPGSQL AS
$$
BEGIN
    INSERT INTO markets_current(id,tx_hash,vega_time,instrument_id,tradable_instrument,decimal_places,fees,opening_auction,price_monitoring_settings,liquidity_monitoring_parameters,trading_mode,state,market_timestamps,position_decimal_places,lp_price_range, linear_slippage_factor, quadratic_slippage_factor, parent_market_id, insurance_pool_fraction)
    VALUES(NEW.id,NEW.tx_hash,NEW.vega_time,NEW.instrument_id,NEW.tradable_instrument,NEW.decimal_places,NEW.fees,NEW.opening_auction,NEW.price_monitoring_settings,NEW.liquidity_monitoring_parameters,NEW.trading_mode,NEW.state,NEW.market_timestamps,NEW.position_decimal_places,NEW.lp_price_range, NEW.linear_slippage_factor, NEW.quadratic_slippage_factor, NEW.parent_market_id, NEW.insurance_pool_fraction)
    ON CONFLICT(id) DO UPDATE SET
                                  tx_hash=EXCLUDED.tx_hash,
                                  instrument_id=EXCLUDED.instrument_id,
                                  tradable_instrument=EXCLUDED.tradable_instrument,
                                  decimal_places=EXCLUDED.decimal_places,
                                  fees=EXCLUDED.fees,
                                  opening_auction=EXCLUDED.opening_auction,
                                  price_monitoring_settings=EXCLUDED.price_monitoring_settings,
                                  liquidity_monitoring_parameters=EXCLUDED.liquidity_monitoring_parameters,
                                  trading_mode=EXCLUDED.trading_mode,
                                  state=EXCLUDED.state,
                                  market_timestamps=EXCLUDED.market_timestamps,
                                  position_decimal_places=EXCLUDED.position_decimal_places,
                                  lp_price_range=EXCLUDED.lp_price_range,
                                  linear_slippage_factor=EXCLUDED.linear_slippage_factor,
                                  quadratic_slippage_factor=EXCLUDED.quadratic_slippage_factor,
                                  vega_time=EXCLUDED.vega_time,
                                  parent_market_id=EXCLUDED.parent_market_id,
                                  insurance_pool_fraction=EXCLUDED.insurance_pool_fraction;
    RETURN NULL;
END;
$$;
-- +goose StatementEnd

CREATE TABLE IF NOT EXISTS market_lineage (
    market_id BYTEA NOT NULL PRIMARY KEY,
    parent_market_id BYTEA, -- the root market in the lineage chain will not have a parent, but all subsequent children will
    root_id BYTEA NOT NULL, -- market id of the first market in the lineage chain
    vega_time TIMESTAMP WITH TIME ZONE NOT NULL
);


-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_market_lineage()
    RETURNS TRIGGER
    LANGUAGE PLPGSQL AS
$$
    DECLARE
        lineage_root bytea;
    BEGIN
        -- We only need to add the successor market to the lineage table if the market
        -- proposal for the new market has been accepted, once accepted, all other successor proposals
        -- will be rejected as markets can only have one parent or successor.
        IF NEW.state != 'STATE_PENDING' THEN
            RETURN NULL;
        END IF;

        -- make sure the record doesn't already exist
        SELECT root_id
        INTO lineage_root
        FROM market_lineage WHERE market_id = NEW.parent_market_id;

        IF lineage_root IS NULL THEN
            -- if the parent market doesn't exist in the lineage chain, then the market is the root of the lineage chain
            lineage_root := NEW.id;
        END IF;

        -- insert the lineage entry
        INSERT INTO market_lineage (market_id, parent_market_id, root_id, vega_time)
        VALUES (NEW.id, NEW.parent_market_id, lineage_root, NEW.vega_time)
        ON CONFLICT (market_id)
        DO NOTHING;

        RETURN NULL;
    END;
$$;
-- +goose StatementEnd

CREATE TRIGGER update_market_lineage AFTER INSERT OR UPDATE ON markets FOR EACH ROW EXECUTE FUNCTION update_market_lineage();

-- +goose Down

DROP TRIGGER IF EXISTS update_market_lineage ON markets;
DROP FUNCTION IF EXISTS update_market_lineage;
DROP TABLE IF EXISTS market_lineage CASCADE;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_current_markets()
    RETURNS TRIGGER
    LANGUAGE PLPGSQL AS
$$
BEGIN
    INSERT INTO markets_current(id,tx_hash,vega_time,instrument_id,tradable_instrument,decimal_places,fees,opening_auction,price_monitoring_settings,liquidity_monitoring_parameters,trading_mode,state,market_timestamps,position_decimal_places,lp_price_range, linear_slippage_factor, quadratic_slippage_factor)
    VALUES(NEW.id,NEW.tx_hash,NEW.vega_time,NEW.instrument_id,NEW.tradable_instrument,NEW.decimal_places,NEW.fees,NEW.opening_auction,NEW.price_monitoring_settings,NEW.liquidity_monitoring_parameters,NEW.trading_mode,NEW.state,NEW.market_timestamps,NEW.position_decimal_places,NEW.lp_price_range, NEW.linear_slippage_factor, NEW.quadratic_slippage_factor)
    ON CONFLICT(id) DO UPDATE SET
                                  tx_hash=EXCLUDED.tx_hash,
                                  instrument_id=EXCLUDED.instrument_id,
                                  tradable_instrument=EXCLUDED.tradable_instrument,
                                  decimal_places=EXCLUDED.decimal_places,
                                  fees=EXCLUDED.fees,
                                  opening_auction=EXCLUDED.opening_auction,
                                  price_monitoring_settings=EXCLUDED.price_monitoring_settings,
                                  liquidity_monitoring_parameters=EXCLUDED.liquidity_monitoring_parameters,
                                  trading_mode=EXCLUDED.trading_mode,
                                  state=EXCLUDED.state,
                                  market_timestamps=EXCLUDED.market_timestamps,
                                  position_decimal_places=EXCLUDED.position_decimal_places,
                                  lp_price_range=EXCLUDED.lp_price_range,
                                  linear_slippage_factor=EXCLUDED.linear_slippage_factor,
                                  quadratic_slippage_factor=EXCLUDED.quadratic_slippage_factor,
                                  vega_time=EXCLUDED.vega_time;
    RETURN NULL;
END;
$$;
-- +goose StatementEnd

ALTER TABLE markets_current
    DROP IF EXISTS parent_market_id,
    DROP IF EXISTS insurance_pool_fraction
;

ALTER TABLE markets
    DROP IF EXISTS parent_market_id,
    DROP IF EXISTS insurance_pool_fraction
;
