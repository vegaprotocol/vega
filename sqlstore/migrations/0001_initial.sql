-- +goose Up
create extension if not exists timescaledb;

create table blocks
(
    vega_time     TIMESTAMP WITH TIME ZONE NOT NULL PRIMARY KEY,
    height        BIGINT                   NOT NULL,
    hash          BYTEA                    NOT NULL
);

create table assets
(
    id             BYTEA NOT NULL PRIMARY KEY,
    name           TEXT NOT NULL UNIQUE,
    symbol         TEXT NOT NULL UNIQUE,
    total_supply   NUMERIC(32, 0),
    decimals       INT,
    quantum        INT,
    source         TEXT,
    erc20_contract TEXT,
    vega_time      TIMESTAMP WITH TIME ZONE NOT NULL REFERENCES blocks (vega_time)
);

create table parties
(
    id        BYTEA NOT NULL PRIMARY KEY,
    vega_time TIMESTAMP WITH TIME ZONE REFERENCES blocks (vega_time)
);

create table accounts
(
    id        SERIAL PRIMARY KEY,
    party_id  BYTEA,
    asset_id  BYTEA                    NOT NULL REFERENCES assets (id),
    market_id BYTEA,
    type      INT,
    vega_time TIMESTAMP WITH TIME ZONE NOT NULL REFERENCES blocks(vega_time),

    UNIQUE(party_id, asset_id, market_id, type)
);

create table balances
(
    account_id INT                      NOT NULL REFERENCES accounts(id),
    vega_time  TIMESTAMP WITH TIME ZONE NOT NULL REFERENCES blocks(vega_time),
    balance    NUMERIC(32, 0)           NOT NULL,

    PRIMARY KEY(vega_time, account_id)
);

create table ledger
(
    id              SERIAL                   PRIMARY KEY,
    account_from_id INT                      NOT NULL REFERENCES accounts(id),
    account_to_id   INT                      NOT NULL REFERENCES accounts(id),
    quantity        NUMERIC(32, 0)           NOT NULL,
    vega_time       TIMESTAMP WITH TIME ZONE NOT NULL REFERENCES blocks(vega_time),
    transfer_time   TIMESTAMP WITH TIME ZONE NOT NULL,
    reference       TEXT,
    type            TEXT
);

CREATE TABLE orders (
    id                BYTEA                     NOT NULL,
    market_id         BYTEA                     NOT NULL,
    party_id          BYTEA                     NOT NULL, -- at some point add REFERENCES parties(id),
    side              SMALLINT                  NOT NULL,
    price             BIGINT                    NOT NULL,
    size              BIGINT                    NOT NULL,
    remaining         BIGINT                    NOT NULL,
    time_in_force     SMALLINT                  NOT NULL,
    type              SMALLINT                  NOT NULL,
    status            SMALLINT                  NOT NULL,
    reference         TEXT,
    reason            SMALLINT,
    version           INT                       NOT NULL,
    batch_id          INT                       NOT NULL,
    pegged_offset     INT,
    pegged_reference  SMALLINT,
    lp_id             BYTEA,
    created_at        TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at        TIMESTAMP WITH TIME ZONE,
    expires_at        TIMESTAMP WITH TIME ZONE,
    vega_time         TIMESTAMP WITH TIME ZONE NOT NULL REFERENCES blocks(vega_time),
    PRIMARY key(vega_time, id, version)
);

-- Orders contains all the historical changes to each order (as of the end of the block),
-- this view contains the *current* state of the latest version each order
--  (e.g. it's unique on order ID)
CREATE VIEW orders_current AS (
  SELECT DISTINCT ON (id) * FROM orders ORDER BY id, version DESC, vega_time DESC
);

-- Manual updates to the order (e.g. user changing price level) increment the 'version'
-- this view contains the current state of each *version* of the order (e.g. it is
-- unique on (order ID, version)
CREATE VIEW orders_current_versions AS (
  SELECT DISTINCT ON (id, version) * FROM orders ORDER BY id, version DESC, vega_time DESC
);

create table trades
(
    vega_time       TIMESTAMP WITH TIME ZONE NOT NULL REFERENCES blocks(vega_time),
    seq_num    BIGINT NOT NULL,
    id     BYTEA NOT NULL,
    market_id BYTEA NOT NULL,
    price     NUMERIC(32, 0),
    size      NUMERIC(32, 0),
    buyer     BYTEA NOT NULL,
    seller    BYTEA NOT NULL,
    aggressor SMALLINT,
    buy_order BYTEA NOT NULL,
    sell_order BYTEA NOT NULL,
    type       SMALLINT NOT NULL,
    buyer_maker_fee NUMERIC(32, 0),
    buyer_infrastructure_fee NUMERIC(32, 0),
    buyer_liquidity_fee NUMERIC(32, 0),
    seller_maker_fee NUMERIC(32, 0),
    seller_infrastructure_fee NUMERIC(32, 0),
    seller_liquidity_fee NUMERIC(32, 0),
    buyer_auction_batch BIGINT,
    seller_auction_batch BIGINT
);

SELECT create_hypertable('trades', 'vega_time', chunk_time_interval => INTERVAL '1 day');
CREATE INDEX ON trades (market_id, vega_time DESC);


CREATE TABLE network_limits (
  vega_time                   TIMESTAMP WITH TIME ZONE NOT NULL PRIMARY KEY REFERENCES blocks(vega_time),
  can_propose_market          BOOLEAN NOT NULL,
  can_propose_asset           BOOLEAN NOT NULL,
  bootstrap_finished          BOOLEAN NOT NULL,
  propose_market_enabled      BOOLEAN NOT NULL,
  propose_asset_enabled       BOOLEAN NOT NULL,
  bootstrap_block_count       INTEGER,
  genesis_loaded              BOOLEAN NOT NULL,
  propose_market_enabled_from TIMESTAMP WITH TIME ZONE NOT NULL,
  propose_asset_enabled_from  TIMESTAMP WITH TIME ZONE NOT NULL
);

-- Create a function that always returns the first non-NULL value:
CREATE OR REPLACE FUNCTION public.first_agg (anyelement, anyelement)
  RETURNS anyelement
  LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE AS
'SELECT $1';

-- Then wrap an aggregate around it:
CREATE AGGREGATE public.first (anyelement) (
  SFUNC    = public.first_agg
, STYPE    = anyelement
, PARALLEL = safe
);

-- Create a function that always returns the last non-NULL value:
CREATE OR REPLACE FUNCTION public.last_agg (anyelement, anyelement)
  RETURNS anyelement
  LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE AS
'SELECT $2';

-- Then wrap an aggregate around it:
CREATE AGGREGATE public.last (anyelement) (
  SFUNC    = public.last_agg
, STYPE    = anyelement
, PARALLEL = safe
);

create type auction_trigger_type as enum('AUCTION_TRIGGER_UNSPECIFIED', 'AUCTION_TRIGGER_BATCH', 'AUCTION_TRIGGER_OPENING', 'AUCTION_TRIGGER_PRICE', 'AUCTION_TRIGGER_LIQUIDITY');
create type market_trading_mode_type as enum('TRADING_MODE_UNSPECIFIED', 'TRADING_MODE_CONTINUOUS', 'TRADING_MODE_BATCH_AUCTION', 'TRADING_MODE_OPENING_AUCTION', 'TRADING_MODE_MONITORING_AUCTION');
create type market_state_type as enum('STATE_UNSPECIFIED', 'STATE_PROPOSED', 'STATE_REJECTED', 'STATE_PENDING', 'STATE_CANCELLED', 'STATE_ACTIVE', 'STATE_SUSPENDED', 'STATE_CLOSED', 'STATE_TRADING_TERMINATED', 'STATE_SETTLED');

create table market_data (
    market bytea not null,
    vega_time timestamp with time zone not null references blocks(vega_time),
    seq_num bigint not null,
    mark_price numeric(32),
    best_bid_price numeric(32),
    best_bid_volume bigint,
    best_offer_price numeric(32),
    best_offer_volume bigint,
    best_static_bid_price numeric(32),
    best_static_bid_volume bigint,
    best_static_offer_price numeric(32),
    best_static_offer_volume bigint,
    mid_price numeric(32),
    static_mid_price numeric(32),
    open_interest bigint,
    auction_end bigint,
    auction_start bigint,
    indicative_price numeric(32),
    indicative_volume bigint,
    market_trading_mode market_trading_mode_type,
    auction_trigger auction_trigger_type,
    extension_trigger auction_trigger_type,
    target_stake numeric(32),
    supplied_stake numeric(32),
    price_monitoring_bounds jsonb,
    market_value_proxy text,
    liquidity_provider_fee_shares jsonb
);

select create_hypertable('market_data', 'vega_time', chunk_time_interval => INTERVAL '1 day');

create index on market_data (market, vega_time);

create or replace view market_data_snapshot as
with cte_market_data_latest(market, vega_time) as (
    select market, max(vega_time)
    from market_data
    group by market
)
select md.market, md.vega_time, seq_num, mark_price, best_bid_price, best_bid_volume, best_offer_price, best_offer_volume,
       best_static_bid_price, best_static_bid_volume, best_static_offer_price, best_static_offer_volume,
       mid_price, static_mid_price, open_interest, auction_end, auction_start, indicative_price, indicative_volume,
       market_trading_mode, auction_trigger, extension_trigger, target_stake, supplied_stake, price_monitoring_bounds,
       market_value_proxy, liquidity_provider_fee_shares
from market_data md
join cte_market_data_latest mx
on md.market = mx.market
and md.vega_time = mx.vega_time
;

CREATE TABLE rewards(
  party_id         BYTEA NOT NULL REFERENCES parties(id),
  asset_id         BYTEA NOT NULL REFERENCES assets(id),
  epoch_id         BIGINT NOT NULL,
  amount           NUMERIC(32, 0),
  percent_of_total FLOAT,
  vega_time        TIMESTAMP WITH TIME ZONE NOT NULL
);

CREATE TABLE delegations(
  party_id         BYTEA NOT NULL, -- REFERENCES parties(id), TODO once parties table is populated
  node_id          BYTEA NOT NULL, -- REFERENCES nodes(id),   TODO once we have node table
  epoch_id         BIGINT NOT NULL,
  amount           NUMERIC(32, 0),
  vega_time        TIMESTAMP WITH TIME ZONE NOT NULL
);

create table if not exists markets (
    id bytea not null,
    vega_time timestamp with time zone not null references blocks(vega_time),
    instrument_id text,
    tradable_instrument jsonb,
    decimal_places int,
    fees jsonb,
    opening_auction jsonb,
    price_monitoring_settings jsonb,
    liquidity_monitoring_parameters jsonb,
    trading_mode market_trading_mode_type,
    state market_state_type,
    market_timestamps jsonb,
    position_decimal_places int,
    primary key (id, vega_time)
);

CREATE TABLE epochs(
  id           BIGINT                   NOT NULL,
  start_time   TIMESTAMP WITH TIME ZONE NOT NULL,
  expire_time  TIMESTAMP WITH TIME ZONE NOT NULL,
  end_time     TIMESTAMP WITH TIME ZONE,
  vega_time    TIMESTAMP WITH TIME ZONE NOT NULL REFERENCES blocks(vega_time),
  PRIMARY KEY(id, vega_time)
);

create type deposit_status as enum('STATUS_UNSPECIFIED', 'STATUS_OPEN', 'STATUS_CANCELLED', 'STATUS_FINALIZED');

create table if not exists deposits (
    id bytea not null,
    status deposit_status not null,
    party_id bytea not null,
    asset bytea not null,
    amount numeric(32, 0),
    tx_hash text not null,
    credited_timestamp timestamp with time zone not null,
    created_timestamp timestamp with time zone not null,
    vega_time timestamp with time zone not null references blocks(vega_time),
    primary key (id, party_id, vega_time)
);

CREATE TYPE proposal_state AS enum('STATE_UNSPECIFIED', 'STATE_FAILED', 'STATE_OPEN', 'STATE_PASSED', 'STATE_REJECTED', 'STATE_DECLINED', 'STATE_ENACTED', 'STATE_WAITING_FOR_NODE_VOTE');
CREATE TYPE proposal_error AS enum('PROPOSAL_ERROR_UNSPECIFIED', 'PROPOSAL_ERROR_CLOSE_TIME_TOO_SOON', 'PROPOSAL_ERROR_CLOSE_TIME_TOO_LATE', 'PROPOSAL_ERROR_ENACT_TIME_TOO_SOON', 'PROPOSAL_ERROR_ENACT_TIME_TOO_LATE', 'PROPOSAL_ERROR_INSUFFICIENT_TOKENS', 'PROPOSAL_ERROR_INVALID_INSTRUMENT_SECURITY', 'PROPOSAL_ERROR_NO_PRODUCT', 'PROPOSAL_ERROR_UNSUPPORTED_PRODUCT', 'PROPOSAL_ERROR_NO_TRADING_MODE', 'PROPOSAL_ERROR_UNSUPPORTED_TRADING_MODE', 'PROPOSAL_ERROR_NODE_VALIDATION_FAILED', 'PROPOSAL_ERROR_MISSING_BUILTIN_ASSET_FIELD', 'PROPOSAL_ERROR_MISSING_ERC20_CONTRACT_ADDRESS', 'PROPOSAL_ERROR_INVALID_ASSET', 'PROPOSAL_ERROR_INCOMPATIBLE_TIMESTAMPS', 'PROPOSAL_ERROR_NO_RISK_PARAMETERS', 'PROPOSAL_ERROR_NETWORK_PARAMETER_INVALID_KEY', 'PROPOSAL_ERROR_NETWORK_PARAMETER_INVALID_VALUE', 'PROPOSAL_ERROR_NETWORK_PARAMETER_VALIDATION_FAILED', 'PROPOSAL_ERROR_OPENING_AUCTION_DURATION_TOO_SMALL', 'PROPOSAL_ERROR_OPENING_AUCTION_DURATION_TOO_LARGE', 'PROPOSAL_ERROR_MARKET_MISSING_LIQUIDITY_COMMITMENT', 'PROPOSAL_ERROR_COULD_NOT_INSTANTIATE_MARKET', 'PROPOSAL_ERROR_INVALID_FUTURE_PRODUCT', 'PROPOSAL_ERROR_MISSING_COMMITMENT_AMOUNT', 'PROPOSAL_ERROR_INVALID_FEE_AMOUNT', 'PROPOSAL_ERROR_INVALID_SHAPE', 'PROPOSAL_ERROR_INVALID_RISK_PARAMETER', 'PROPOSAL_ERROR_MAJORITY_THRESHOLD_NOT_REACHED', 'PROPOSAL_ERROR_PARTICIPATION_THRESHOLD_NOT_REACHED', 'PROPOSAL_ERROR_INVALID_ASSET_DETAILS', 'PROPOSAL_ERROR_UNKNOWN_TYPE', 'PROPOSAL_ERROR_UNKNOWN_RISK_PARAMETER_TYPE', 'PROPOSAL_ERROR_INVALID_FREEFORM', 'PROPOSAL_ERROR_INSUFFICIENT_EQUITY_LIKE_SHARE', 'PROPOSAL_ERROR_INVALID_MARKET');
CREATE TYPE vote_value AS enum('VALUE_UNSPECIFIED', 'VALUE_NO', 'VALUE_YES');

CREATE TABLE proposals(
  id                   BYTEA NOT NULL,
  reference            TEXT NOT NULL,
  party_id             BYTEA NOT NULL,  -- TODO, once parties is properly populated REFERENCES parties(id),
  state                proposal_state NOT NULL,
  terms JSONB          NOT NULL,
  reason               proposal_error,
  error_details        TEXT,
  vega_time            TIMESTAMP WITH TIME ZONE NOT NULL REFERENCES blocks(vega_time),
  proposal_time        TIMESTAMP WITH TIME ZONE,
  PRIMARY KEY (id, vega_time)
);

CREATE VIEW proposals_current AS (
  SELECT DISTINCT ON (id) * FROM proposals ORDER BY id, vega_time DESC
);

CREATE TABLE votes(
  proposal_id                    BYTEA                    NOT NULL, -- TODO think about this REFERENCES proposals(id),
  party_id                       BYTEA                    NOT NULL, -- TODO, once parties is properly populated REFERENCES parties(id),
  value                          vote_value               NOT NULL,
  total_governance_token_balance NUMERIC(32, 0)           NOT NULL,
  total_governance_token_weight  NUMERIC(32, 16)           NOT NULL,
  total_equity_like_share_weight NUMERIC(32, 16)           NOT NULL,
  vega_time                      TIMESTAMP WITH TIME ZONE NOT NULL REFERENCES blocks(vega_time),
  PRIMARY KEY (proposal_id, party_id, vega_time)
);

CREATE VIEW votes_current AS (
  SELECT DISTINCT ON (proposal_id, party_id) * FROM votes ORDER BY proposal_id, party_id, vega_time DESC
);

create table if not exists margin_levels (
    market_id bytea not null,
    asset_id bytea not null,
    party_id bytea not null,
    timestamp timestamp with time zone not null,
    maintenance_margin numeric(32, 0),
    search_level numeric(32, 0),
    initial_margin numeric(32, 0),
    collateral_release_level numeric(32, 0),
    vega_time timestamp with time zone not null references blocks(vega_time),
    primary key (market_id, asset_id, party_id, vega_time)
);

create table if not exists risk_factors (
    market_id bytea not null,
    short numeric(32, 16) not null,
    long numeric(32, 16) not null,
    vega_time timestamp with time zone not null references blocks(vega_time),
    primary key (market_id, vega_time)
);

-- +goose Down
DROP AGGREGATE IF EXISTS public.first(anyelement);
DROP AGGREGATE IF EXISTS public.last(anyelement);
DROP FUNCTION IF EXISTS public.first_agg(anyelement, anyelement);
DROP FUNCTION IF EXISTS public.last_agg(anyelement, anyelement);

DROP VIEW IF EXISTS votes_current;
DROP TABLE IF EXISTS votes;
DROP VIEW IF EXISTS proposals_current;
DROP TABLE IF EXISTS proposals;
DROP TYPE IF EXISTS vote_value;
DROP TYPE IF EXISTS proposal_error;
DROP TYPE IF EXISTS proposal_state;

DROP TABLE IF EXISTS epochs;
DROP TABLE IF EXISTS delegations;
DROP TABLE IF EXISTS rewards;

DROP TABLE IF EXISTS network_limits;
DROP VIEW IF EXISTS orders_current;
DROP VIEW IF EXISTS orders_current_versions;

drop table if exists risk_factors;
drop table if exists margin_levels;

DROP TABLE IF EXISTS deposits;
DROP TYPE IF EXISTS deposit_status;

DROP TABLE IF EXISTS orders;
DROP TYPE IF EXISTS order_time_in_force;
DROP TYPE IF EXISTS order_status;
DROP TYPE IF EXISTS order_side;
DROP TYPE IF EXISTS order_type;
DROP TYPE IF EXISTS order_pegged_reference;

DROP TABLE IF EXISTS markets;
DROP VIEW IF EXISTS market_data_snapshot;
DROP TABLE IF EXISTS market_data;
DROP TYPE IF EXISTS auction_trigger_type;
DROP TYPE IF EXISTS market_trading_mode_type;
DROP TYPE IF EXISTS market_state_type;

DROP TABLE IF EXISTS ledger;
DROP TABLE IF EXISTS balances;
DROP TABLE IF EXISTS accounts;
DROP TABLE IF EXISTS parties;
DROP TABLE IF EXISTS assets;
DROP TABLE IF EXISTS trades;
DROP TABLE IF EXISTS blocks cascade;

