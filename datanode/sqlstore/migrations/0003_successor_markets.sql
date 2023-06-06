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

ALTER TYPE proposal_error ADD VALUE IF NOT EXISTS 'PROPOSAL_ERROR_INVALID_SUCCESSOR_MARKET';

-- +goose Down

-- We cannot just drop a value from an enum so we have to create a new type
CREATE TYPE proposal_error_new AS enum('PROPOSAL_ERROR_UNSPECIFIED', 'PROPOSAL_ERROR_CLOSE_TIME_TOO_SOON', 'PROPOSAL_ERROR_CLOSE_TIME_TOO_LATE', 'PROPOSAL_ERROR_ENACT_TIME_TOO_SOON', 'PROPOSAL_ERROR_ENACT_TIME_TOO_LATE', 'PROPOSAL_ERROR_INSUFFICIENT_TOKENS', 'PROPOSAL_ERROR_INVALID_INSTRUMENT_SECURITY', 'PROPOSAL_ERROR_NO_PRODUCT', 'PROPOSAL_ERROR_UNSUPPORTED_PRODUCT', 'PROPOSAL_ERROR_NO_TRADING_MODE', 'PROPOSAL_ERROR_UNSUPPORTED_TRADING_MODE', 'PROPOSAL_ERROR_NODE_VALIDATION_FAILED', 'PROPOSAL_ERROR_MISSING_BUILTIN_ASSET_FIELD', 'PROPOSAL_ERROR_MISSING_ERC20_CONTRACT_ADDRESS', 'PROPOSAL_ERROR_INVALID_ASSET', 'PROPOSAL_ERROR_INCOMPATIBLE_TIMESTAMPS', 'PROPOSAL_ERROR_NO_RISK_PARAMETERS', 'PROPOSAL_ERROR_NETWORK_PARAMETER_INVALID_KEY', 'PROPOSAL_ERROR_NETWORK_PARAMETER_INVALID_VALUE', 'PROPOSAL_ERROR_NETWORK_PARAMETER_VALIDATION_FAILED', 'PROPOSAL_ERROR_OPENING_AUCTION_DURATION_TOO_SMALL', 'PROPOSAL_ERROR_OPENING_AUCTION_DURATION_TOO_LARGE', 'PROPOSAL_ERROR_MARKET_MISSING_LIQUIDITY_COMMITMENT', 'PROPOSAL_ERROR_COULD_NOT_INSTANTIATE_MARKET', 'PROPOSAL_ERROR_INVALID_FUTURE_PRODUCT', 'PROPOSAL_ERROR_MISSING_COMMITMENT_AMOUNT', 'PROPOSAL_ERROR_INVALID_FEE_AMOUNT', 'PROPOSAL_ERROR_INVALID_SHAPE', 'PROPOSAL_ERROR_INVALID_RISK_PARAMETER', 'PROPOSAL_ERROR_MAJORITY_THRESHOLD_NOT_REACHED', 'PROPOSAL_ERROR_PARTICIPATION_THRESHOLD_NOT_REACHED', 'PROPOSAL_ERROR_INVALID_ASSET_DETAILS', 'PROPOSAL_ERROR_UNKNOWN_TYPE', 'PROPOSAL_ERROR_UNKNOWN_RISK_PARAMETER_TYPE', 'PROPOSAL_ERROR_INVALID_FREEFORM', 'PROPOSAL_ERROR_INSUFFICIENT_EQUITY_LIKE_SHARE', 'PROPOSAL_ERROR_INVALID_MARKET', 'PROPOSAL_ERROR_TOO_MANY_MARKET_DECIMAL_PLACES', 'PROPOSAL_ERROR_TOO_MANY_PRICE_MONITORING_TRIGGERS', 'PROPOSAL_ERROR_ERC20_ADDRESS_ALREADY_IN_USE');

-- Delete anything that was using the enum value we are dropping because if we have to roll back then the data should be invalid too
DELETE FROM proposals WHERE reason = 'PROPOSAL_ERROR_INVALID_SUCCESSOR_MARKET';

-- Temporarily drop the proposals_current view so we can drop the enum
DROP VIEW IF EXISTS proposals_current;

-- Change the table to use the new enum without the value we're dropping
ALTER TABLE proposals ALTER COLUMN reason TYPE proposal_error_new USING reason::text::proposal_error_new;

-- Drop the old enum which contains the value(s) we need to remove
DROP TYPE proposal_error;

-- Rename the new enum to the old name
ALTER TYPE proposal_error_new RENAME TO proposal_error;

-- Recreate the proposals_current view
CREATE VIEW proposals_current AS (
    SELECT DISTINCT ON (id) * FROM proposals ORDER BY id, vega_time DESC
);


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
