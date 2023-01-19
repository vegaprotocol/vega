-- +goose Up
create extension if not exists timescaledb;

CREATE DOMAIN HUGEINT AS NUMERIC(1000, 0);

create table blocks
(
    vega_time     TIMESTAMP WITH TIME ZONE NOT NULL PRIMARY KEY,
    height        BIGINT                   NOT NULL,
    hash          BYTEA                    NOT NULL
);
select create_hypertable('blocks', 'vega_time', chunk_time_interval => INTERVAL '1 day');
create index on blocks (height);

create table last_block
(
    onerow_check  bool PRIMARY KEY DEFAULT TRUE,
    vega_time     TIMESTAMP WITH TIME ZONE NOT NULL,
    height        BIGINT                   NOT NULL,
    hash          BYTEA                    NOT NULL
);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_last_block()
    RETURNS TRIGGER
    LANGUAGE PLPGSQL AS
$$
BEGIN
    Insert into last_block (vega_time, height, hash) VALUES(NEW.vega_time, NEW.height, NEW.hash) on conflict(onerow_check) do update
    set
        vega_time=EXCLUDED.vega_time,
        height=EXCLUDED.height,
        hash=EXCLUDED.hash;
    RETURN NULL;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER update_last_block AFTER INSERT ON blocks FOR EACH ROW EXECUTE function update_last_block();





create table chain
(
    id            TEXT NOT NULL,
    onerow_check  bool PRIMARY KEY DEFAULT TRUE
);

create type asset_status_type as enum('STATUS_UNSPECIFIED', 'STATUS_PROPOSED', 'STATUS_REJECTED', 'STATUS_PENDING_LISTING', 'STATUS_ENABLED');

create table assets
(
    id                  BYTEA NOT NULL,
    name                TEXT NOT NULL,
    symbol              TEXT NOT NULL,
    decimals            INT,
    quantum             HUGEINT,
    source              TEXT,
    erc20_contract      TEXT,
    lifetime_limit      HUGEINT NOT NULL,
    withdraw_threshold  HUGEINT NOT NULL,
    status		asset_status_type NOT NULL,
    tx_hash             BYTEA NOT NULL,
    vega_time           TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY (id, vega_time)
);

CREATE VIEW assets_current AS (
  SELECT DISTINCT ON (id) * FROM assets ORDER BY id, vega_time DESC
);

create table parties
(
    id        BYTEA NOT NULL PRIMARY KEY,
    tx_hash   BYTEA NOT NULL,
    vega_time TIMESTAMP WITH TIME ZONE
);

create table accounts
(
    id        BYTEA PRIMARY KEY,
    party_id  BYTEA,
    asset_id  BYTEA  NOT NULL,
    market_id BYTEA,
    type      INT,
    tx_hash   BYTEA NOT NULL,
    vega_time TIMESTAMP WITH TIME ZONE NOT NULL,
    UNIQUE(party_id, asset_id, market_id, type)
);

create table balances
(
    account_id bytea                      NOT NULL,
    vega_time  TIMESTAMP WITH TIME ZONE NOT NULL,
    tx_hash    BYTEA NOT NULL,
    balance    HUGEINT           NOT NULL,
    PRIMARY KEY(vega_time, account_id) INCLUDE (balance)
);

select create_hypertable('balances', 'vega_time', chunk_time_interval => INTERVAL '1 day');
create index on balances (account_id, vega_time) INCLUDE(balance);

create table current_balances
(
    account_id bytea                      NOT NULL,
    tx_hash    BYTEA                    NOT NULL,
    vega_time  TIMESTAMP WITH TIME ZONE NOT NULL,
    balance    HUGEINT           NOT NULL,

    PRIMARY KEY(account_id)
);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_current_balances()
   RETURNS TRIGGER
   LANGUAGE PLPGSQL AS
$$
    BEGIN
    INSERT INTO current_balances(account_id, tx_hash, vega_time, balance) VALUES(NEW.account_id, NEW.tx_hash, NEW.vega_time, NEW.balance)
      ON CONFLICT(account_id) DO UPDATE SET
         balance=EXCLUDED.balance,
         tx_hash=EXCLUDED.tx_hash,
         vega_time=EXCLUDED.vega_time;
    RETURN NULL;
    END;
$$;
-- +goose StatementEnd

CREATE TRIGGER update_current_balances AFTER INSERT ON balances FOR EACH ROW EXECUTE function update_current_balances();

CREATE MATERIALIZED VIEW conflated_balances
            WITH (timescaledb.continuous, timescaledb.materialized_only = true) AS
SELECT account_id, time_bucket('1 hour', vega_time) AS bucket,
       last(balance, vega_time) AS balance,
       last(tx_hash, vega_time) AS tx_hash,
       last(vega_time, vega_time) AS vega_time
FROM balances
GROUP BY account_id, bucket WITH NO DATA;

-- start_offset is set to a day, as data is append only this does not impact the processing time and ensures
-- that the CAGG data will be correct on recovery in the event of a transient outage ( < 1 day )
SELECT add_continuous_aggregate_policy('conflated_balances', start_offset => INTERVAL '1 day',
                                     end_offset => INTERVAL '1 hour', schedule_interval => INTERVAL '1 hour');

CREATE VIEW all_balances AS
(
SELECT
    balances.account_id,
    balances.tx_hash,
    balances.vega_time,
    balances.balance
FROM balances
UNION ALL
SELECT
    conflated_balances.account_id,
    conflated_balances.tx_hash,
    conflated_balances.vega_time,
    conflated_balances.balance
FROM conflated_balances
WHERE conflated_balances.vega_time < (SELECT coalesce(min(balances.vega_time), 'infinity') FROM balances));

create table ledger
(
    ledger_entry_time              TIMESTAMP WITH TIME ZONE NOT NULL,
    account_from_id                bytea                    NOT NULL,
    account_to_id                  bytea                    NOT NULL,
    quantity                       HUGEINT                  NOT NULL,
    tx_hash                        BYTEA                    NOT NULL,
    vega_time                      TIMESTAMP WITH TIME ZONE NOT NULL,
    transfer_time                  TIMESTAMP WITH TIME ZONE NOT NULL,
    account_from_balance  HUGEINT                  NOT NULL,
    account_to_balance    HUGEINT                  NOT NULL,
    type                           TEXT,
    PRIMARY KEY(ledger_entry_time)
);
SELECT create_hypertable('ledger', 'ledger_entry_time', chunk_time_interval => INTERVAL '1 day');

CREATE INDEX ON ledger (account_from_id, vega_time DESC);
CREATE INDEX ON ledger (account_to_id, vega_time DESC);
CREATE INDEX ON ledger (type, vega_time DESC);

DROP TABLE IF EXISTS orders_history;

CREATE TABLE orders (
    id                BYTEA                     NOT NULL,
    market_id         BYTEA                     NOT NULL,
    party_id          BYTEA                     NOT NULL, -- at some point add REFERENCES parties(id),
    side              SMALLINT                  NOT NULL,
    price             HUGEINT                    NOT NULL,
    size              BIGINT                    NOT NULL,
    remaining         BIGINT                    NOT NULL,
    time_in_force     SMALLINT                  NOT NULL,
    type              SMALLINT                  NOT NULL,
    status            SMALLINT                  NOT NULL,
    reference         TEXT,
    reason            SMALLINT,
    version           INT                       NOT NULL,
    batch_id          INT                       NOT NULL,
    pegged_offset     HUGEINT,
    pegged_reference  SMALLINT,
    lp_id             BYTEA,
    created_at        TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at        TIMESTAMP WITH TIME ZONE,
    expires_at        TIMESTAMP WITH TIME ZONE,
    tx_hash           BYTEA                    NOT NULL,
    vega_time         TIMESTAMP WITH TIME ZONE NOT NULL,
    seq_num           BIGINT NOT NULL,
    vega_time_to      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT 'infinity',
    PRIMARY KEY(vega_time, seq_num)
);

SELECT create_hypertable('orders', 'vega_time', chunk_time_interval => INTERVAL '1 day');
CREATE INDEX ON orders (market_id, vega_time DESC) where vega_time_to='infinity';
CREATE INDEX ON orders (party_id, vega_time DESC) where vega_time_to='infinity';
CREATE INDEX ON orders (reference, vega_time DESC) where vega_time_to='infinity';
CREATE INDEX ON orders (id, vega_time_to);

CREATE TABLE orders_live (
    id                BYTEA                     NOT NULL,
    market_id         BYTEA                     NOT NULL,
    party_id          BYTEA                     NOT NULL, -- at some point add REFERENCES parties(id),
    side              SMALLINT                  NOT NULL,
    price             HUGEINT                   NOT NULL,
    size              BIGINT                    NOT NULL,
    remaining         BIGINT                    NOT NULL,
    time_in_force     SMALLINT                  NOT NULL,
    type              SMALLINT                  NOT NULL,
    status            SMALLINT                  NOT NULL,
    reference         TEXT,
    reason            SMALLINT,
    version           INT                       NOT NULL,
    batch_id          INT                       NOT NULL,
    pegged_offset     HUGEINT,
    pegged_reference  SMALLINT,
    lp_id             BYTEA,
    created_at        TIMESTAMP WITH TIME ZONE NOT NULL,
    updated_at        TIMESTAMP WITH TIME ZONE,
    expires_at        TIMESTAMP WITH TIME ZONE,
    tx_hash           BYTEA                    NOT NULL,
    vega_time         TIMESTAMP WITH TIME ZONE NOT NULL,
    seq_num           BIGINT NOT NULL, -- event sequence number in the block
    vega_time_to      TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT 'infinity',
    PRIMARY KEY(id)
);

CREATE INDEX ON orders_live (market_id, vega_time DESC);
CREATE INDEX ON orders_live (party_id, vega_time DESC);
CREATE INDEX ON orders_live (reference, vega_time DESC);
CREATE INDEX ON orders_live USING HASH (id);

-- +goose StatementBegin

CREATE OR REPLACE FUNCTION archive_orders()
   RETURNS TRIGGER
   LANGUAGE PLPGSQL AS
$$
    BEGIN
    -- It is permitted by core to re-use order IDs and 'resurrect' done orders (specifically,
    -- LP orders do this, so we need to check our history table to see if we need to updated
    -- vega_time_to on any most-recent-version-of an order.
    UPDATE orders
       SET vega_time_to = NEW.vega_time
     WHERE vega_time_to = 'infinity'
       AND id = NEW.id;

      DELETE from orders_live
        WHERE id = NEW.id;

    -- As per https://github.com/vegaprotocol/specs-internal/blob/master/protocol/0024-OSTA-order_status.md
    -- we consider an order 'live' if it either ACTIVE (status=1) or PARKED (status=8). Orders
    -- with statuses other than this are discarded by core, so we consider them candidates for
    -- eventual deletion according to the data retention policy by placing them in orders_history.
    IF NEW.status IN (1, 8)
    THEN
       INSERT INTO orders_live
       VALUES(new.id, new.market_id, new.party_id, new.side, new.price,
              new.size, new.remaining, new.time_in_force, new.type, new.status,
              new.reference, new.reason, new.version, new.batch_id, new.pegged_offset,
              new.pegged_reference, new.lp_id, new.created_at, new.updated_at, new.expires_at,
              new.tx_hash, new.vega_time, new.seq_num, 'infinity');
    END IF;

    RETURN NEW;

    END;
$$;

-- +goose StatementEnd

CREATE TRIGGER archive_orders BEFORE INSERT ON orders FOR EACH ROW EXECUTE function archive_orders();


-- Orders contains all the historical changes to each order (as of the end of the block),
-- this view contains the *current* state of the latest version each order
--  (e.g. it's unique on order ID)
CREATE VIEW orders_current AS (
  SELECT * FROM orders WHERE vega_time_to = 'infinity'
);

-- Manual updates to the order (e.g. user changing price level) increment the 'version'
-- this view contains the current state of each *version* of the order (e.g. it is
-- unique on (order ID, version)
CREATE VIEW orders_current_versions AS (
  SELECT DISTINCT ON (id, version) * FROM orders ORDER BY id, version DESC, vega_time DESC
);

create table trades
(
    synthetic_time       TIMESTAMP WITH TIME ZONE NOT NULL,
    tx_hash              BYTEA                    NOT NULL,
    vega_time       TIMESTAMP WITH TIME ZONE NOT NULL,
    seq_num    BIGINT NOT NULL,
    id     BYTEA NOT NULL,
    market_id BYTEA NOT NULL,
    price     HUGEINT NOT NULL,
    size      BIGINT NOT NULL,
    buyer     BYTEA NOT NULL,
    seller    BYTEA NOT NULL,
    aggressor SMALLINT,
    buy_order BYTEA NOT NULL,
    sell_order BYTEA NOT NULL,
    type       SMALLINT NOT NULL,
    buyer_maker_fee HUGEINT,
    buyer_infrastructure_fee HUGEINT,
    buyer_liquidity_fee HUGEINT,
    seller_maker_fee HUGEINT,
    seller_infrastructure_fee HUGEINT,
    seller_liquidity_fee HUGEINT,
    buyer_auction_batch BIGINT,
    seller_auction_batch BIGINT,
    primary key (synthetic_time)
);

SELECT create_hypertable('trades', 'synthetic_time', chunk_time_interval => INTERVAL '1 day');
CREATE INDEX ON trades (market_id, synthetic_time DESC);
CREATE INDEX ON trades(buyer, synthetic_time desc);
CREATE INDEX ON trades(seller, synthetic_time desc);

CREATE MATERIALIZED VIEW trades_candle_1_minute
            WITH (timescaledb.continuous) AS
SELECT market_id, time_bucket('1 minute', synthetic_time) AS period_start,
       first(price, synthetic_time) AS open,
       last(price, synthetic_time) AS close,
       max(price) AS high,
       min(price) AS low,
       sum(size) AS volume,
       last(synthetic_time,
            synthetic_time) AS last_update_in_period
FROM trades
GROUP BY market_id, period_start WITH NO DATA;

-- start_offset is set to a day, as data is append only this does not impact the processing time and ensures
-- that the CAGG data will be correct on recovery in the event of a transient outage ( < 1 day )
SELECT add_continuous_aggregate_policy('trades_candle_1_minute', start_offset => INTERVAL '1 day', end_offset => INTERVAL '1 minute', schedule_interval => INTERVAL '1 minute');

CREATE MATERIALIZED VIEW trades_candle_5_minutes
            WITH (timescaledb.continuous) AS
SELECT market_id, time_bucket('5 minutes', synthetic_time) AS period_start,
       first(price, synthetic_time) AS open,
       last(price, synthetic_time) AS close,
       max(price) AS high,
       min(price) AS low,
       sum(size) AS volume,
       last(synthetic_time,
            synthetic_time) AS last_update_in_period
FROM trades
GROUP BY market_id, period_start WITH NO DATA;

-- start_offset is set to a day, as data is append only this does not impact the processing time and ensures
-- that the CAGG data will be correct on recovery in the event of a transient outage ( < 1 day )
SELECT add_continuous_aggregate_policy('trades_candle_5_minutes', start_offset => INTERVAL '1 day', end_offset => INTERVAL '5 minutes', schedule_interval => INTERVAL '5 minutes');

CREATE MATERIALIZED VIEW trades_candle_15_minutes
            WITH (timescaledb.continuous) AS
SELECT market_id, time_bucket('15 minutes', synthetic_time) AS period_start,
       first(price, synthetic_time) AS open,
       last(price, synthetic_time) AS close,
       max(price) AS high,
       min(price) AS low,
       sum(size) AS volume,
       last(synthetic_time,
            synthetic_time) AS last_update_in_period
FROM trades
GROUP BY market_id, period_start WITH NO DATA;

-- start_offset is set to a day, as data is append only this does not impact the processing time and ensures
-- that the CAGG data will be correct on recovery in the event of a transient outage ( < 1 day )
SELECT add_continuous_aggregate_policy('trades_candle_15_minutes', start_offset => INTERVAL '1 day', end_offset => INTERVAL '15 minutes', schedule_interval => INTERVAL '15 minutes');

CREATE MATERIALIZED VIEW trades_candle_1_hour
            WITH (timescaledb.continuous) AS
SELECT market_id, time_bucket('1 hour', synthetic_time) AS period_start,
       first(price, synthetic_time) AS open,
       last(price, synthetic_time) AS close,
       max(price) AS high,
       min(price) AS low,
       sum(size) AS volume,
       last(synthetic_time,
            synthetic_time) AS last_update_in_period
FROM trades
GROUP BY market_id, period_start WITH NO DATA;

-- start_offset is set to a day, as data is append only this does not impact the processing time and ensures
-- that the CAGG data will be correct on recovery in the event of a transient outage ( < 1 day )
SELECT add_continuous_aggregate_policy('trades_candle_1_hour', start_offset => INTERVAL '1 day', end_offset => INTERVAL '1 hour', schedule_interval => INTERVAL '1 hour');

CREATE MATERIALIZED VIEW trades_candle_6_hours
            WITH (timescaledb.continuous) AS
SELECT market_id, time_bucket('6 hours', synthetic_time) AS period_start,
       first(price, synthetic_time) AS open,
       last(price, synthetic_time) AS close,
       max(price) AS high,
       min(price) AS low,
       sum(size) AS volume,
       last(synthetic_time,
            synthetic_time) AS last_update_in_period
FROM trades
GROUP BY market_id, period_start WITH NO DATA;

-- start_offset is set to a day, as data is append only this does not impact the processing time and ensures
-- that the CAGG data will be correct on recovery in the event of a transient outage ( < 1 day )
SELECT add_continuous_aggregate_policy('trades_candle_6_hours', start_offset => INTERVAL '1 day', end_offset => INTERVAL '6 hours', schedule_interval => INTERVAL '6 hours');

CREATE MATERIALIZED VIEW trades_candle_1_day
            WITH (timescaledb.continuous) AS
SELECT market_id, time_bucket('1 day', synthetic_time) AS period_start,
       first(price, synthetic_time) AS open,
       last(price, synthetic_time) AS close,
       max(price) AS high,
       min(price) AS low,
       sum(size) AS volume,
       last(synthetic_time,
            synthetic_time) AS last_update_in_period
FROM trades
GROUP BY market_id, period_start WITH NO DATA;

SELECT add_continuous_aggregate_policy('trades_candle_1_day', start_offset => INTERVAL '3 days', end_offset => INTERVAL '1 day', schedule_interval => INTERVAL '1 day');


CREATE TABLE network_limits (
  vega_time                   TIMESTAMP WITH TIME ZONE NOT NULL PRIMARY KEY,
  tx_hash                     BYTEA                    NOT NULL,
  can_propose_market          BOOLEAN NOT NULL,
  can_propose_asset           BOOLEAN NOT NULL,
  propose_market_enabled      BOOLEAN NOT NULL,
  propose_asset_enabled       BOOLEAN NOT NULL,
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
create type market_trading_mode_type as enum('TRADING_MODE_UNSPECIFIED', 'TRADING_MODE_CONTINUOUS', 'TRADING_MODE_BATCH_AUCTION', 'TRADING_MODE_OPENING_AUCTION', 'TRADING_MODE_MONITORING_AUCTION', 'TRADING_MODE_NO_TRADING');
create type market_state_type as enum('STATE_UNSPECIFIED', 'STATE_PROPOSED', 'STATE_REJECTED', 'STATE_PENDING', 'STATE_CANCELLED', 'STATE_ACTIVE', 'STATE_SUSPENDED', 'STATE_CLOSED', 'STATE_TRADING_TERMINATED', 'STATE_SETTLED');

create table market_data (
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
    PRIMARY KEY (synthetic_time)
);

select create_hypertable('market_data', 'synthetic_time', chunk_time_interval => INTERVAL '1 day');

create index on market_data (market, vega_time);

create or replace view market_data_snapshot as
with cte_market_data_latest(market, vega_time) as (
    select market, max(vega_time)
    from market_data
    group by market
)
select md.market, md.tx_hash, md.vega_time, seq_num, mark_price, best_bid_price, best_bid_volume, best_offer_price, best_offer_volume,
       best_static_bid_price, best_static_bid_volume, best_static_offer_price, best_static_offer_volume,
       mid_price, static_mid_price, open_interest, auction_end, auction_start, indicative_price, indicative_volume,
       market_trading_mode, auction_trigger, extension_trigger, target_stake, supplied_stake, price_monitoring_bounds,
       market_value_proxy, liquidity_provider_fee_shares, market_state
from market_data md
join cte_market_data_latest mx
on md.market = mx.market
and md.vega_time = mx.vega_time
;

CREATE TYPE node_status as enum('NODE_STATUS_UNSPECIFIED', 'NODE_STATUS_VALIDATOR', 'NODE_STATUS_NON_VALIDATOR');

CREATE TABLE IF NOT EXISTS nodes (
  id                    BYTEA NOT NULL,
  vega_pub_key          BYTEA NOT NULL,
  tendermint_pub_key    BYTEA NOT NULL,
  ethereum_address      BYTEA NOT NULL,
  info_url              TEXT NOT NULL,
  location              TEXT NOT NULL,
  status                node_status NOT NULL,
  name                  TEXT NOT NULL,
  avatar_url            TEXT NOT NULL,
  tx_hash               BYTEA NOT NULL,
  vega_time             TIMESTAMP WITH TIME ZONE NOT NULL,
  PRIMARY KEY(id)
);


CREATE TABLE IF NOT EXISTS nodes_announced (
  node_id               BYTEA NOT NULL,
  epoch_seq             BIGINT NOT NULL,
  added                 BOOLEAN NOT NULL,
  tx_hash               BYTEA NOT NULL,
  vega_time             TIMESTAMP WITH TIME ZONE NOT NULL,
  PRIMARY KEY(node_id, epoch_seq, vega_time)
);

CREATE TYPE validator_node_status as enum(
  'VALIDATOR_NODE_STATUS_UNSPECIFIED',
  'VALIDATOR_NODE_STATUS_TENDERMINT',
  'VALIDATOR_NODE_STATUS_ERSATZ',
  'VALIDATOR_NODE_STATUS_PENDING'
);

CREATE TABLE IF NOT EXISTS ranking_scores (
  node_id           BYTEA NOT NULL REFERENCES nodes(id),
  epoch_seq         BIGINT NOT NULL,

  stake_score       NUMERIC NOT NULL,
  performance_score NUMERIC NOT NULL,
  ranking_score     NUMERIC NOT NULL,
  voting_power      INT NOT NULL,

  previous_status   validator_node_status NOT NULL,
  status            validator_node_status NOT NULL,

  tx_hash            BYTEA NOT NULL,
  vega_time         TIMESTAMP WITH TIME ZONE NOT NULL,

  PRIMARY KEY (node_id, epoch_seq)
);

CREATE TABLE IF NOT EXISTS reward_scores (
  node_id                 BYTEA NOT NULL REFERENCES nodes(id),
  epoch_seq               BIGINT NOT NULL,

  validator_node_status   validator_node_status NOT NULL,

  raw_validator_score     NUMERIC NOT NULL,
  performance_score       NUMERIC NOT NULL,
  multisig_score          NUMERIC NOT NULL,
  validator_score         NUMERIC NOT NULL,
  normalised_score        NUMERIC NOT NULL,

  tx_hash                 BYTEA NOT NULL,
  vega_time               TIMESTAMP WITH TIME ZONE NOT NULL,

  PRIMARY KEY (node_id, epoch_seq)
);

CREATE TABLE rewards(
  party_id         BYTEA NOT NULL REFERENCES parties(id),
  asset_id         BYTEA NOT NULL,
  market_id        BYTEA NOT NULL,
  reward_type      TEXT NOT NULL,
  epoch_id         BIGINT NOT NULL,
  amount           HUGEINT,
  percent_of_total FLOAT,
  timestamp        TIMESTAMP WITH TIME ZONE NOT NULL,
  tx_hash          BYTEA NOT NULL,
  vega_time        TIMESTAMP WITH TIME ZONE NOT NULL,
  seq_num           BIGINT NOT NULL,
  primary key (vega_time, seq_num)
);

create index on rewards (party_id, asset_id);
create index on rewards (asset_id);
create index on rewards (epoch_id);

CREATE TABLE delegations(
  party_id         BYTEA NOT NULL,
  node_id          BYTEA NOT NULL,
  epoch_id         BIGINT NOT NULL,
  amount           HUGEINT,
  tx_hash          BYTEA NOT NULL,
  vega_time        TIMESTAMP WITH TIME ZONE NOT NULL,
  seq_num  BIGINT NOT NULL,
  PRIMARY KEY(vega_time, seq_num)
);

select create_hypertable('delegations', 'vega_time', chunk_time_interval => INTERVAL '1 day');
create index on delegations (party_id, node_id, epoch_id);


create table delegations_current
(
    party_id         BYTEA NOT NULL,
    node_id          BYTEA NOT NULL,
    epoch_id         BIGINT NOT NULL,
    amount           HUGEINT,
    tx_hash          BYTEA NOT NULL,
    vega_time        TIMESTAMP WITH TIME ZONE NOT NULL,
    seq_num  BIGINT NOT NULL,
    primary key (party_id, node_id, epoch_id)
);

create index on delegations_current(node_id, epoch_id);
create index on delegations_current(epoch_id);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_current_delegations()
    RETURNS TRIGGER
    LANGUAGE PLPGSQL AS
$$
BEGIN
    INSERT INTO delegations_current(party_id,node_id,epoch_id,amount,tx_hash,vega_time,seq_num)
    VALUES(NEW.party_id,NEW.node_id,NEW.epoch_id,NEW.amount,NEW.tx_hash,NEW.vega_time,NEW.seq_num)
    ON CONFLICT(party_id, node_id, epoch_id) DO UPDATE SET
                                                           amount=EXCLUDED.amount,
                                                           tx_hash=EXCLUDED.tx_hash,
                                                           vega_time=EXCLUDED.vega_time,
                                                           seq_num=EXCLUDED.seq_num;
    RETURN NULL;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER update_current_delegations AFTER INSERT ON delegations FOR EACH ROW EXECUTE function update_current_delegations();


create table if not exists markets (
    id bytea not null,
    tx_hash bytea not null,
    vega_time timestamp with time zone not null,
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
    lp_price_range text,
    primary key (id, vega_time)
);

select create_hypertable('markets', 'vega_time', chunk_time_interval => INTERVAL '1 day');

drop view if exists markets_current;

create table if not exists markets_current (
    id bytea not null,
    tx_hash bytea not null,
    vega_time timestamp with time zone not null,
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
    lp_price_range text,
    primary key (id)
);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_current_markets()
    RETURNS TRIGGER
    LANGUAGE PLPGSQL AS
$$
BEGIN
    INSERT INTO markets_current(id,tx_hash,vega_time,instrument_id,tradable_instrument,decimal_places,fees,opening_auction,price_monitoring_settings,liquidity_monitoring_parameters,trading_mode,state,market_timestamps,position_decimal_places,lp_price_range)
    VALUES(NEW.id,NEW.tx_hash,NEW.vega_time,NEW.instrument_id,NEW.tradable_instrument,NEW.decimal_places,NEW.fees,NEW.opening_auction,NEW.price_monitoring_settings,NEW.liquidity_monitoring_parameters,NEW.trading_mode,NEW.state,NEW.market_timestamps,NEW.position_decimal_places,NEW.lp_price_range)
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
                                                           vega_time=EXCLUDED.vega_time;
    RETURN NULL;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER update_current_markets
    AFTER INSERT OR UPDATE ON markets
    FOR EACH ROW EXECUTE function update_current_markets();

CREATE TABLE epochs(
  id           BIGINT                   NOT NULL,
  start_time   TIMESTAMP WITH TIME ZONE NOT NULL,
  expire_time  TIMESTAMP WITH TIME ZONE NOT NULL,
  end_time     TIMESTAMP WITH TIME ZONE,
  tx_hash      BYTEA                    NOT NULL,
  vega_time    TIMESTAMP WITH TIME ZONE NOT NULL,
  PRIMARY KEY(id, vega_time)
);

create type deposit_status as enum('STATUS_UNSPECIFIED', 'STATUS_OPEN', 'STATUS_CANCELLED', 'STATUS_FINALIZED');

create table if not exists deposits (
    id bytea not null,
    status deposit_status not null,
    party_id bytea not null,
    asset bytea not null,
    amount HUGEINT,
    foreign_tx_hash text not null,
    credited_timestamp timestamp with time zone not null,
    created_timestamp timestamp with time zone not null,
    tx_hash  bytea not null,
    vega_time timestamp with time zone not null,
    primary key (id, party_id, vega_time)
);
CREATE INDEX ON deposits(party_id);

select create_hypertable('deposits', 'vega_time', chunk_time_interval => INTERVAL '1 day');

CREATE VIEW deposits_current AS (
    -- Assume that party_id is always the same for a given deposit ID to allow filter to be pushed down
    SELECT DISTINCT ON (id, party_id) * FROM deposits ORDER BY id, party_id, vega_time DESC
);

create type withdrawal_status as enum('STATUS_UNSPECIFIED', 'STATUS_OPEN', 'STATUS_REJECTED', 'STATUS_FINALIZED');

create table if not exists withdrawals (
    id bytea not null,
    party_id bytea not null,
    amount numeric,
    asset bytea not null,
    status withdrawal_status not null,
    ref text not null,
    foreign_tx_hash text not null,
    created_timestamp timestamp with time zone not null,
    withdrawn_timestamp timestamp with time zone not null,
    ext jsonb not null,
    tx_hash  bytea not null,
    vega_time timestamp with time zone not null,
    primary key (id, party_id, vega_time)
);

CREATE INDEX ON withdrawals(party_id);

select create_hypertable('withdrawals', 'vega_time', chunk_time_interval => INTERVAL '1 day');

CREATE VIEW withdrawals_current AS (
    -- Assume that party_id is always the same for a given withdrawal ID to allow filter to be pushed down
    SELECT DISTINCT ON (id, party_id) * FROM withdrawals ORDER BY id, party_id, vega_time DESC
);

CREATE TYPE proposal_state AS enum('STATE_UNSPECIFIED', 'STATE_FAILED', 'STATE_OPEN', 'STATE_PASSED', 'STATE_REJECTED', 'STATE_DECLINED', 'STATE_ENACTED', 'STATE_WAITING_FOR_NODE_VOTE');
CREATE TYPE proposal_error AS enum('PROPOSAL_ERROR_UNSPECIFIED', 'PROPOSAL_ERROR_CLOSE_TIME_TOO_SOON', 'PROPOSAL_ERROR_CLOSE_TIME_TOO_LATE', 'PROPOSAL_ERROR_ENACT_TIME_TOO_SOON', 'PROPOSAL_ERROR_ENACT_TIME_TOO_LATE', 'PROPOSAL_ERROR_INSUFFICIENT_TOKENS', 'PROPOSAL_ERROR_INVALID_INSTRUMENT_SECURITY', 'PROPOSAL_ERROR_NO_PRODUCT', 'PROPOSAL_ERROR_UNSUPPORTED_PRODUCT', 'PROPOSAL_ERROR_NO_TRADING_MODE', 'PROPOSAL_ERROR_UNSUPPORTED_TRADING_MODE', 'PROPOSAL_ERROR_NODE_VALIDATION_FAILED', 'PROPOSAL_ERROR_MISSING_BUILTIN_ASSET_FIELD', 'PROPOSAL_ERROR_MISSING_ERC20_CONTRACT_ADDRESS', 'PROPOSAL_ERROR_INVALID_ASSET', 'PROPOSAL_ERROR_INCOMPATIBLE_TIMESTAMPS', 'PROPOSAL_ERROR_NO_RISK_PARAMETERS', 'PROPOSAL_ERROR_NETWORK_PARAMETER_INVALID_KEY', 'PROPOSAL_ERROR_NETWORK_PARAMETER_INVALID_VALUE', 'PROPOSAL_ERROR_NETWORK_PARAMETER_VALIDATION_FAILED', 'PROPOSAL_ERROR_OPENING_AUCTION_DURATION_TOO_SMALL', 'PROPOSAL_ERROR_OPENING_AUCTION_DURATION_TOO_LARGE', 'PROPOSAL_ERROR_MARKET_MISSING_LIQUIDITY_COMMITMENT', 'PROPOSAL_ERROR_COULD_NOT_INSTANTIATE_MARKET', 'PROPOSAL_ERROR_INVALID_FUTURE_PRODUCT', 'PROPOSAL_ERROR_MISSING_COMMITMENT_AMOUNT', 'PROPOSAL_ERROR_INVALID_FEE_AMOUNT', 'PROPOSAL_ERROR_INVALID_SHAPE', 'PROPOSAL_ERROR_INVALID_RISK_PARAMETER', 'PROPOSAL_ERROR_MAJORITY_THRESHOLD_NOT_REACHED', 'PROPOSAL_ERROR_PARTICIPATION_THRESHOLD_NOT_REACHED', 'PROPOSAL_ERROR_INVALID_ASSET_DETAILS', 'PROPOSAL_ERROR_UNKNOWN_TYPE', 'PROPOSAL_ERROR_UNKNOWN_RISK_PARAMETER_TYPE', 'PROPOSAL_ERROR_INVALID_FREEFORM', 'PROPOSAL_ERROR_INSUFFICIENT_EQUITY_LIKE_SHARE', 'PROPOSAL_ERROR_INVALID_MARKET', 'PROPOSAL_ERROR_TOO_MANY_MARKET_DECIMAL_PLACES', 'PROPOSAL_ERROR_TOO_MANY_PRICE_MONITORING_TRIGGERS', 'PROPOSAL_ERROR_ERC20_ADDRESS_ALREADY_IN_USE');
CREATE TYPE vote_value AS enum('VALUE_UNSPECIFIED', 'VALUE_NO', 'VALUE_YES');


CREATE TABLE proposals(
  id                        BYTEA NOT NULL,
  reference                 TEXT NOT NULL,
  party_id                  BYTEA NOT NULL,  -- TODO, once parties is properly populated REFERENCES parties(id),
  state                     proposal_state NOT NULL,
  terms                     JSONB          NOT NULL,
  rationale                 JSONB          NOT NULL,
  reason                    proposal_error,
  error_details             TEXT,
  vega_time                 TIMESTAMP WITH TIME ZONE NOT NULL,
  proposal_time             TIMESTAMP WITH TIME ZONE,
  required_majority         NUMERIC(1000, 16) NOT NULL,
  required_participation    NUMERIC(1000, 16) NOT NULL,
  required_lp_majority      NUMERIC(1000, 16),
  required_lp_participation NUMERIC(1000, 16),
  tx_hash                   BYTEA NOT NULL,
  PRIMARY KEY (id, vega_time)
);

CREATE VIEW proposals_current AS (
  SELECT DISTINCT ON (id) * FROM proposals ORDER BY id, vega_time DESC
);

CREATE TABLE votes(
  proposal_id                    BYTEA                    NOT NULL, -- TODO think about this REFERENCES proposals(id),
  party_id                       BYTEA                    NOT NULL, -- TODO, once parties is properly populated REFERENCES parties(id),
  value                          vote_value               NOT NULL,
  total_governance_token_balance HUGEINT           NOT NULL,
  total_governance_token_weight  NUMERIC(1000, 16)           NOT NULL,
  total_equity_like_share_weight NUMERIC(1000, 16)           NOT NULL,
  tx_hash                        BYTEA NOT NULL,
  vega_time                      TIMESTAMP WITH TIME ZONE NOT NULL,
  initial_time                   TIMESTAMP WITH TIME ZONE,
  PRIMARY KEY (proposal_id, party_id, vega_time)
);

CREATE INDEX ON votes(party_id);

CREATE VIEW votes_current AS (
  SELECT DISTINCT ON (proposal_id, party_id) * FROM votes ORDER BY proposal_id, party_id, vega_time DESC
);

create table if not exists margin_levels (
    account_id bytea NOT NULL,
    timestamp timestamp with time zone not null,
    maintenance_margin HUGEINT,
    search_level HUGEINT,
    initial_margin HUGEINT,
    collateral_release_level HUGEINT,
    tx_hash BYTEA NOT NULL,
    vega_time timestamp with time zone not null,
    PRIMARY KEY(vega_time, account_id)
);

select create_hypertable('margin_levels', 'vega_time', chunk_time_interval => INTERVAL '1 day');

create table current_margin_levels
(
    account_id bytea NOT NULL,
    timestamp timestamp with time zone not null,
    maintenance_margin HUGEINT,
    search_level HUGEINT,
    initial_margin HUGEINT,
    collateral_release_level HUGEINT,
    tx_hash BYTEA NOT NULL,
    vega_time timestamp with time zone not null,

    PRIMARY KEY(account_id)
);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_current_margin_levels()
   RETURNS TRIGGER
   LANGUAGE PLPGSQL AS
$$
BEGIN
INSERT INTO current_margin_levels(account_id,
                                  timestamp,
                                  maintenance_margin,
                                  search_level,
                                  initial_margin,
                                  collateral_release_level,
                                  tx_hash,
                                  vega_time) VALUES(NEW.account_id,
                                                    NEW.timestamp,
                                                    NEW.maintenance_margin,
                                                    NEW.search_level,
                                                    NEW.initial_margin,
                                                    NEW.collateral_release_level,
                                                    NEW.tx_hash,
                                                    NEW.vega_time)
    ON CONFLICT(account_id) DO UPDATE SET
                                   timestamp=EXCLUDED.timestamp,
                                   maintenance_margin=EXCLUDED.maintenance_margin,
                                   search_level=EXCLUDED.search_level,
                                   initial_margin=EXCLUDED.initial_margin,
                                   collateral_release_level=EXCLUDED.collateral_release_level,
                                   tx_hash=EXCLUDED.tx_hash,
                                   vega_time=EXCLUDED.vega_time;
RETURN NULL;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER update_current_margin_levels AFTER INSERT ON margin_levels FOR EACH ROW EXECUTE function update_current_margin_levels();



CREATE MATERIALIZED VIEW conflated_margin_levels
            WITH (timescaledb.continuous, timescaledb.materialized_only = true) AS
SELECT account_id, time_bucket('1 minute', vega_time) AS bucket,
       last(maintenance_margin, vega_time) AS maintenance_margin,
       last(search_level, vega_time) AS search_level,
       last(initial_margin, vega_time) AS initial_margin,
       last(collateral_release_level, vega_time) AS collateral_release_level,
       last(timestamp, vega_time) AS timestamp,
       last(tx_hash, vega_time) AS tx_hash,
       last(vega_time, vega_time) AS vega_time
FROM margin_levels
GROUP BY account_id, bucket WITH NO DATA;

-- start_offset is set to a day, as data is append only this does not impact the processing time and ensures
-- that the CAGG data will be correct on recovery in the event of a transient outage ( < 1 day )
SELECT add_continuous_aggregate_policy('conflated_margin_levels', start_offset => INTERVAL '1 day',
    end_offset => INTERVAL '1 minute', schedule_interval => INTERVAL '1 minute');

CREATE VIEW all_margin_levels AS
(
SELECT margin_levels.account_id,
       margin_levels."timestamp",
       margin_levels.maintenance_margin,
       margin_levels.search_level,
       margin_levels.initial_margin,
       margin_levels.collateral_release_level,
       margin_levels.tx_hash,
       margin_levels.vega_time
FROM margin_levels
UNION ALL
SELECT conflated_margin_levels.account_id,
       conflated_margin_levels."timestamp",
       conflated_margin_levels.maintenance_margin,
       conflated_margin_levels.search_level,
       conflated_margin_levels.initial_margin,
       conflated_margin_levels.collateral_release_level,
       conflated_margin_levels.tx_hash,
       conflated_margin_levels.vega_time
FROM conflated_margin_levels
WHERE conflated_margin_levels.vega_time < (SELECT coalesce(min(margin_levels.vega_time), 'infinity') FROM margin_levels));

create table if not exists risk_factors (
    market_id bytea not null,
    short NUMERIC(1000, 16) not null,
    long NUMERIC(1000, 16) not null,
    tx_hash bytea not null,
    vega_time timestamp with time zone not null,
    primary key (market_id, vega_time)
);

CREATE VIEW risk_factors_current AS (
    SELECT DISTINCT ON (market_id) * FROM risk_factors ORDER BY market_id, vega_time DESC
);

CREATE TABLE network_parameters (
    key          TEXT                     NOT NULL,
    value        TEXT                     NOT NULL,
    tx_hash      BYTEA                    NOT NULL,
    vega_time    TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY (key, vega_time)
);

drop view if exists network_parameters_current;

CREATE TABLE network_parameters_current (
    key          TEXT                     NOT NULL,
    value        TEXT                     NOT NULL,
    tx_hash      BYTEA                    NOT NULL,
    vega_time    TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY (key)
);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_current_network_parameters()
    RETURNS TRIGGER
    LANGUAGE PLPGSQL AS
$$
BEGIN
    INSERT INTO network_parameters_current(key, value, tx_hash, vega_time)
    VALUES(NEW.key, NEW.value, NEW.tx_hash, NEW.vega_time)
    ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value, tx_hash=EXCLUDED.tx_hash, vega_time=EXCLUDED.vega_time;
    RETURN NULL;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER update_current_network_parameters AFTER INSERT OR UPDATE ON network_parameters
    FOR EACH ROW EXECUTE function update_current_network_parameters();

CREATE TABLE checkpoints(
    hash         TEXT                     NOT NULL,
    block_hash   TEXT                     NOT NULL,
    block_height BIGINT                   NOT NULL,
    tx_hash      BYTEA                    NOT NULL,
    vega_time    TIMESTAMP WITH TIME ZONE NOT NULL,
    seq_num  BIGINT NOT NULL,
    PRIMARY KEY(vega_time, seq_num)
);

SELECT create_hypertable('checkpoints', 'vega_time', chunk_time_interval => INTERVAL '1 day');

CREATE TABLE positions(
  market_id           BYTEA NOT NULL,
  party_id            BYTEA NOT NULL,
  open_volume         BIGINT NOT NULL,
  realised_pnl        NUMERIC NOT NULL,
  unrealised_pnl      NUMERIC NOT NULL,
  average_entry_price NUMERIC NOT NULL,
  average_entry_market_price NUMERIC NOT NULL,
  loss                NUMERIC NOT NULL,
  adjustment          NUMERIC NOT NULL,
  tx_hash             BYTEA                    NOT NULL,
  vega_time           TIMESTAMP WITH TIME ZONE NOT NULL,
  primary key (party_id, market_id, vega_time)
);

select create_hypertable('positions', 'vega_time', chunk_time_interval => INTERVAL '1 day');


CREATE MATERIALIZED VIEW conflated_positions
            WITH (timescaledb.continuous, timescaledb.materialized_only = true) AS
SELECT market_id, party_id, time_bucket('1 hour', vega_time) AS bucket,
 last(open_volume, vega_time) AS open_volume,
 last(realised_pnl, vega_time) AS realised_pnl,
 last(unrealised_pnl, vega_time) AS unrealised_pnl,
 last(average_entry_price, vega_time) AS average_entry_price,
 last(average_entry_market_price, vega_time) AS average_entry_market_price,
 last(loss, vega_time) AS loss,
 last(adjustment, vega_time) AS adjustment,
 last(tx_hash, vega_time) AS tx_hash,
 last(vega_time, vega_time) AS vega_time
FROM positions
GROUP BY market_id, party_id, bucket WITH NO DATA;

-- start_offset is set to a day, as data is append only this does not impact the processing time and ensures
-- that the CAGG data will be correct on recovery in the event of a transient outage ( < 1 day )
SELECT add_continuous_aggregate_policy('conflated_positions', start_offset => INTERVAL '1 day',
                                       end_offset => INTERVAL '1 hour', schedule_interval => INTERVAL '1 hour');

CREATE VIEW all_positions AS
(
SELECT
  positions.market_id,
  positions.party_id,
  positions.open_volume,
  positions.realised_pnl,
  positions.unrealised_pnl,
  positions.average_entry_price,
  positions.average_entry_market_price,
  positions.loss,
  positions.adjustment,
  positions.tx_hash,
  positions.vega_time
FROM positions
UNION ALL
SELECT
    conflated_positions.market_id,
    conflated_positions.party_id,
    conflated_positions.open_volume,
    conflated_positions.realised_pnl,
    conflated_positions.unrealised_pnl,
    conflated_positions.average_entry_price,
    conflated_positions.average_entry_market_price,
    conflated_positions.loss,
    conflated_positions.adjustment,
    conflated_positions.tx_hash,
    conflated_positions.vega_time
FROM conflated_positions
WHERE conflated_positions.vega_time < (SELECT coalesce(min(positions.vega_time), 'infinity') FROM positions));

drop view if exists positions_current;

create table positions_current
(
    market_id           BYTEA NOT NULL,
    party_id            BYTEA NOT NULL,
    open_volume         BIGINT NOT NULL,
    realised_pnl        NUMERIC NOT NULL,
    unrealised_pnl      NUMERIC NOT NULL,
    average_entry_price NUMERIC NOT NULL,
    average_entry_market_price NUMERIC NOT NULL,
    loss                NUMERIC NOT NULL,
    adjustment          NUMERIC NOT NULL,
    tx_hash             BYTEA                    NOT NULL,
    vega_time           TIMESTAMP WITH TIME ZONE NOT NULL,
    primary key (party_id, market_id)

);

CREATE INDEX ON positions_current(market_id);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_current_positions()
    RETURNS TRIGGER
    LANGUAGE PLPGSQL AS
$$
BEGIN
    INSERT INTO positions_current(market_id,party_id,open_volume,realised_pnl,unrealised_pnl,average_entry_price,average_entry_market_price,loss,adjustment,tx_hash,vega_time)
    VALUES(NEW.market_id,NEW.party_id,NEW.open_volume,NEW.realised_pnl,NEW.unrealised_pnl,NEW.average_entry_price,NEW.average_entry_market_price,NEW.loss,NEW.adjustment,NEW.tx_hash,NEW.vega_time)
    ON CONFLICT(party_id, market_id) DO UPDATE SET
                                                   open_volume=EXCLUDED.open_volume,
                                                   realised_pnl=EXCLUDED.realised_pnl,
                                                   unrealised_pnl=EXCLUDED.unrealised_pnl,
                                                   average_entry_price=EXCLUDED.average_entry_price,
                                                   average_entry_market_price=EXCLUDED.average_entry_market_price,
                                                   loss=EXCLUDED.loss,
                                                   adjustment=EXCLUDED.adjustment,
                                                   tx_hash=EXCLUDED.tx_hash,
                                                   vega_time=EXCLUDED.vega_time;
    RETURN NULL;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER update_current_positions AFTER INSERT ON positions FOR EACH ROW EXECUTE function update_current_positions();

create type oracle_spec_status as enum('STATUS_UNSPECIFIED', 'STATUS_ACTIVE', 'STATUS_DEACTIVATED');

create table if not exists oracle_specs (
    id bytea not null,
    created_at timestamp with time zone not null,
    updated_at timestamp with time zone not null,
    signers bytea[],
    filters jsonb,
    status oracle_spec_status not null,
    tx_hash  bytea not null,
    vega_time timestamp with time zone not null,
    primary key (id, vega_time)
);

create table if not exists oracle_data (
    signers bytea[],
    data jsonb not null,
    matched_spec_ids bytea[],
    broadcast_at timestamp with time zone not null,
    tx_hash  bytea not null,
    vega_time timestamp with time zone not null,
    seq_num  BIGINT NOT NULL,
    PRIMARY KEY(vega_time, seq_num)
);

create index if not exists idx_oracle_data_matched_spec_ids on oracle_data(matched_spec_ids);

drop view if exists oracle_data_current;

create table if not exists oracle_data_current (
    signers bytea[],
    data jsonb not null,
    matched_spec_ids bytea[],
    broadcast_at timestamp with time zone not null,
    tx_hash  bytea not null,
    vega_time timestamp with time zone not null,
    seq_num  BIGINT NOT NULL,
    PRIMARY KEY(matched_spec_ids, data)
);

-- +goose StatementBegin

CREATE OR REPLACE FUNCTION update_current_oracle_data()
    RETURNS TRIGGER
    LANGUAGE PLPGSQL AS
$$
BEGIN
    INSERT INTO oracle_data_current(signers,data,matched_spec_ids,broadcast_at,tx_hash,vega_time,seq_num)
    VALUES(NEW.signers,NEW.data,NEW.matched_spec_ids,NEW.broadcast_at,NEW.tx_hash,NEW.vega_time,NEW.seq_num)
    ON CONFLICT(matched_spec_ids, data) DO UPDATE SET
                                                   signers=EXCLUDED.signers,
                                                   broadcast_at=EXCLUDED.broadcast_at,
                                                   tx_hash=EXCLUDED.tx_hash,
                                                   vega_time=EXCLUDED.vega_time,
                                                   seq_num=EXCLUDED.seq_num;
    RETURN NULL;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER update_current_oracle_data AFTER INSERT ON oracle_data FOR EACH ROW EXECUTE function update_current_oracle_data();

create type liquidity_provision_status as enum('STATUS_UNSPECIFIED', 'STATUS_ACTIVE', 'STATUS_STOPPED',
    'STATUS_CANCELLED', 'STATUS_REJECTED', 'STATUS_UNDEPLOYED', 'STATUS_PENDING');

create table if not exists liquidity_provisions (
    id bytea not null,
    party_id bytea,
    created_at timestamp with time zone not null,
    updated_at timestamp with time zone not null,
    market_id bytea,
    commitment_amount HUGEINT,
    fee NUMERIC(1000, 16),
    sells jsonb,
    buys jsonb,
    version bigint,
    status liquidity_provision_status not null,
    reference text,
    tx_hash bytea not null,
    vega_time timestamp with time zone not null,
    primary key (id, vega_time)
);

select create_hypertable('liquidity_provisions', 'vega_time', chunk_time_interval => INTERVAL '1 day');


create table current_liquidity_provisions
(
    id bytea not null,
    party_id bytea,
    created_at timestamp with time zone not null,
    updated_at timestamp with time zone not null,
    market_id bytea,
    commitment_amount HUGEINT,
    fee NUMERIC(1000, 16),
    sells jsonb,
    buys jsonb,
    version bigint,
    status liquidity_provision_status not null,
    reference text,
    tx_hash bytea not null,
    vega_time timestamp with time zone not null,
    primary key (id)
);

create index on current_liquidity_provisions (party_id);
create index on current_liquidity_provisions (market_id, party_id);
create index on current_liquidity_provisions (reference);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_current_liquidity_provisions()
   RETURNS TRIGGER
   LANGUAGE PLPGSQL AS
$$
BEGIN
INSERT INTO current_liquidity_provisions(id,
                             party_id,
                             created_at,
                             updated_at,
                             market_id,
                             commitment_amount,
                             fee,
                             sells,
                             buys,
                             version,
                             status,
                             reference,
                             tx_hash,
                             vega_time
                             ) VALUES(NEW.id,
                                      NEW.party_id,
                                      NEW.created_at,
                                      NEW.updated_at,
                                      NEW.market_id,
                                      NEW.commitment_amount,
                                      NEW.fee,
                                      NEW.sells,
                                      NEW.buys,
                                      NEW.version,
                                      NEW.status,
                                      NEW.reference,
                                      NEW.tx_hash,
                                      NEW.vega_time)
    ON CONFLICT(id) DO UPDATE SET
    party_id=EXCLUDED.party_id,
   created_at=EXCLUDED.created_at,
   updated_at=EXCLUDED.updated_at,
   market_id=EXCLUDED.market_id,
   commitment_amount=EXCLUDED.commitment_amount,
   fee=EXCLUDED.fee,
   sells=EXCLUDED.sells,
   buys=EXCLUDED.buys,
   version=EXCLUDED.version,
   status=EXCLUDED.status,
   reference=EXCLUDED.reference,
   tx_hash=EXCLUDED.tx_hash,
   vega_time=EXCLUDED.vega_time;
RETURN NULL;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER update_current_liquidity_provisions AFTER INSERT ON liquidity_provisions FOR EACH ROW EXECUTE function update_current_liquidity_provisions();



CREATE TYPE transfer_type AS enum('OneOff','Recurring','Unknown');
CREATE TYPE transfer_status AS enum('STATUS_UNSPECIFIED','STATUS_PENDING','STATUS_DONE','STATUS_REJECTED','STATUS_STOPPED','STATUS_CANCELLED');

create table if not exists transfers (
         id bytea not null,
         tx_hash bytea not null,
         vega_time timestamp with time zone not null,
         from_account_id bytea NOT NULL REFERENCES accounts(id),
         to_account_id bytea NOT NULL REFERENCES accounts(id),
         asset_id bytea not null,
         amount        HUGEINT           NOT NULL,
         reference       TEXT,
         status           transfer_status NOT NULL,
         transfer_type   transfer_type NOT NULL,
         deliver_on      TIMESTAMP WITH TIME ZONE,
         start_epoch     BIGINT,
         end_epoch       BIGINT,
         factor        NUMERIC(1000, 16) ,
         dispatch_metric INT,
         dispatch_metric_asset TEXT,
         dispatch_markets TEXT[],
         reason TEXT,
         primary key (id, vega_time)
);

create index on transfers (from_account_id);
create index on transfers (to_account_id);

-- Assume that from/to account is never changed for a given xfer id
CREATE VIEW transfers_current AS ( SELECT DISTINCT ON (id, from_account_id, to_account_id) * FROM transfers ORDER BY id, from_account_id, to_account_id, vega_time DESC);

create table if not exists key_rotations (
  node_id bytea not null references nodes(id),
  old_pub_key bytea not null,
  new_pub_key bytea not null,
  block_height bigint not null,
  tx_hash bytea not null,
  vega_time timestamp with time zone not null,

  primary key (node_id, vega_time)
);

create table if not exists ethereum_key_rotations (
  node_id bytea not null references nodes(id),
  old_address bytea not null,
  new_address bytea not null,
  block_height bigint not null,
  tx_hash bytea not null,
  vega_time timestamp with time zone not null,
  seq_num           BIGINT NOT NULL,
  primary key (seq_num, vega_time)
);

create type erc20_multisig_signer_event as enum('SIGNER_ADDED', 'SIGNER_REMOVED');

create table if not exists erc20_multisig_signer_events(
    id bytea not null,
    validator_id bytea not null,
    signer_change bytea not null,
    submitter bytea not null,
    nonce text not null,
    event erc20_multisig_signer_event not null,
    tx_hash bytea not null,
    vega_time timestamp with time zone,
    epoch_id bigint not null,
    primary key (id)
);

create type stake_linking_type as enum('TYPE_UNSPECIFIED', 'TYPE_LINK', 'TYPE_UNLINK');
create type stake_linking_status as enum('STATUS_UNSPECIFIED', 'STATUS_PENDING', 'STATUS_ACCEPTED', 'STATUS_REJECTED');

create table if not exists stake_linking(
    id bytea not null,
    stake_linking_type stake_linking_type not null,
    ethereum_timestamp timestamp with time zone not null,
    party_id bytea not null,
    amount HUGEINT,
    stake_linking_status stake_linking_status not null,
    finalized_at timestamp with time zone,
    foreign_tx_hash text not null,
    foreign_block_height bigint,
    foreign_block_time bigint,
    log_index bigint,
    ethereum_address text not null,
    tx_hash bytea not null,
    vega_time timestamp with time zone not null,
    primary key (id, vega_time)
);

drop view if exists stake_linking_current;

create table if not exists stake_linking_current(
    id bytea not null,
    stake_linking_type stake_linking_type not null,
    ethereum_timestamp timestamp with time zone not null,
    party_id bytea not null,
    amount HUGEINT,
    stake_linking_status stake_linking_status not null,
    finalized_at timestamp with time zone,
    foreign_tx_hash text not null,
    foreign_block_height bigint,
    foreign_block_time bigint,
    log_index bigint,
    ethereum_address text not null,
    tx_hash bytea not null,
    vega_time timestamp with time zone not null,
    primary key (id)
);

create index on stake_linking_current(party_id);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_current_stake_linking()
    RETURNS TRIGGER
    LANGUAGE plpgsql AS
$$
BEGIN
    INSERT INTO stake_linking_current (id, stake_linking_type, ethereum_timestamp, party_id, amount, stake_linking_status, finalized_at, foreign_tx_hash, foreign_block_height, foreign_block_time, log_index, ethereum_address, tx_hash, vega_time)
    VALUES (NEW.id,
            NEW.stake_linking_type,
            NEW.ethereum_timestamp,
            NEW.party_id,
            NEW.amount,
            NEW.stake_linking_status,
            NEW.finalized_at,
            NEW.foreign_tx_hash,
            NEW.foreign_block_height,
            NEW.foreign_block_time,
            NEW.log_index,
            NEW.ethereum_address,
            NEW.tx_hash,
            NEW.vega_time)
    ON CONFLICT(id) DO UPDATE SET
    stake_linking_type=EXCLUDED.stake_linking_type,
    ethereum_timestamp=EXCLUDED.ethereum_timestamp,
    party_id=EXCLUDED.party_id,
    amount=EXCLUDED.amount,
    stake_linking_status=EXCLUDED.stake_linking_status,
    finalized_at=EXCLUDED.finalized_at,
    foreign_tx_hash=EXCLUDED.foreign_tx_hash,
    foreign_block_height=EXCLUDED.foreign_block_height,
    foreign_block_time=EXCLUDED.foreign_block_time,
    log_index=EXCLUDED.log_index,
    ethereum_address=EXCLUDED.ethereum_address,
    tx_hash=EXCLUDED.tx_hash,
    vega_time=EXCLUDED.vega_time;
    RETURN NULL;
END;
$$;
-- +goose StatementEnd

create trigger update_current_stake_linking
    after insert or update on stake_linking
    for each row execute procedure update_current_stake_linking();

create type node_signature_kind as enum('NODE_SIGNATURE_KIND_UNSPECIFIED', 'NODE_SIGNATURE_KIND_ASSET_NEW', 'NODE_SIGNATURE_KIND_ASSET_WITHDRAWAL', 'NODE_SIGNATURE_KIND_ERC20_MULTISIG_SIGNER_ADDED', 'NODE_SIGNATURE_KIND_ERC20_MULTISIG_SIGNER_REMOVED', 'NODE_SIGNATURE_KIND_ASSET_UPDATE');

create table if not exists node_signatures(
    resource_id bytea not null,
    sig bytea not null,
    kind node_signature_kind,
    tx_hash bytea not null,
    vega_time timestamp with time zone not null,
    primary key (resource_id, sig)
);

CREATE TYPE protocol_upgrade_proposal_status AS enum(
    'PROTOCOL_UPGRADE_PROPOSAL_STATUS_UNSPECIFIED',
    'PROTOCOL_UPGRADE_PROPOSAL_STATUS_PENDING',
    'PROTOCOL_UPGRADE_PROPOSAL_STATUS_APPROVED',
    'PROTOCOL_UPGRADE_PROPOSAL_STATUS_REJECTED');

CREATE TABLE IF NOT EXISTS protocol_upgrade_proposals(
    upgrade_block_height BIGINT NOT NULL,
    vega_release_tag TEXT NOT NULL,
    approvers TEXT[] NOT NULL,
    status protocol_upgrade_proposal_status NOT NULL,
    tx_hash bytea not null,
    vega_time TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY(vega_time, upgrade_block_height, vega_release_tag)
);

CREATE VIEW protocol_upgrade_proposals_current AS (
    SELECT DISTINCT ON (upgrade_block_height, vega_release_tag) *
      FROM protocol_upgrade_proposals
  ORDER BY upgrade_block_height, vega_release_tag, vega_time DESC);

CREATE TABLE IF NOT EXISTS core_snapshots(
    block_height BIGINT NOT NULL,
    block_hash TEXT null,
    vega_core_version TEXT null,
    tx_hash bytea not null,
    vega_time TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY(vega_time, block_height)
);

create table last_snapshot_span
(
    onerow_check  bool PRIMARY KEY DEFAULT TRUE,
    from_height        BIGINT                   NOT NULL,
    to_height          BIGINT                    NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS last_snapshot_span;

DROP AGGREGATE IF EXISTS public.first(anyelement);
DROP AGGREGATE IF EXISTS public.last(anyelement);
DROP FUNCTION IF EXISTS public.first_agg(anyelement, anyelement);
DROP FUNCTION IF EXISTS public.last_agg(anyelement, anyelement);

DROP VIEW IF EXISTS protocol_upgrade_proposals_current;
DROP TABLE IF EXISTS protocol_upgrade_proposals;
DROP TYPE IF EXISTS protocol_upgrade_proposal_status;
DROP TABLE IF EXISTS core_snapshots;
DROP TABLE IF EXISTS ethereum_key_rotations;

DROP TABLE IF EXISTS key_rotations;

DROP VIEW IF EXISTS transfers_current;
DROP TABLE IF EXISTS transfers;
DROP TYPE IF EXISTS transfer_status;
DROP TYPE IF EXISTS transfer_type;


DROP TABLE IF EXISTS checkpoints;

drop trigger if exists update_current_network_parameters on network_parameters;
drop function if exists update_current_network_parameters;
drop table if exists network_parameters_current;
DROP TABLE IF EXISTS network_parameters cascade;

drop trigger if exists update_current_stake_linking on stake_linking;
drop function if exists update_current_stake_linking;
DROP TABLE IF EXISTS stake_linking_current;
DROP TABLE IF EXISTS stake_linking cascade;
DROP TYPE IF EXISTS stake_linking_status;
DROP TYPE IF EXISTS stake_linking_type;

DROP TABLE IF EXISTS node_signatures;
DROP TYPE IF EXISTS node_signature_kind;

DROP TABLE IF EXISTS liquidity_provisions;
DROP TRIGGER IF EXISTS update_current_liquidity_provisions ON liquidity_provisions;
DROP FUNCTION IF EXISTS update_current_liquidity_provisions;
DROP TABLE IF EXISTS current_liquidity_provisions;
DROP TYPE IF EXISTS liquidity_provision_status;

drop trigger if exists update_current_oracle_data on oracle_data;
drop function if exists update_current_oracle_data;
DROP TABLE IF EXISTS oracle_data_current;
DROP INDEX IF EXISTS idx_oracle_data_matched_spec_ids;
DROP TABLE IF EXISTS oracle_data cascade;
DROP TABLE IF EXISTS oracle_specs;
DROP TYPE IF EXISTS oracle_spec_status;

DROP TABLE IF EXISTS positions_current;
DROP TABLE IF EXISTS positions cascade;
DROP TRIGGER IF EXISTS update_current_positions ON positions;
DROP FUNCTION IF EXISTS update_current_positions;

DROP VIEW IF EXISTS votes_current;
DROP TABLE IF EXISTS votes;
DROP VIEW IF EXISTS proposals_current;
DROP TABLE IF EXISTS proposals;
DROP TYPE IF EXISTS vote_value;
DROP TYPE IF EXISTS proposal_error;
DROP TYPE IF EXISTS proposal_state;

DROP TABLE IF EXISTS epochs;

DROP TRIGGER IF EXISTS update_current_delegations ON delegations;
DROP FUNCTION IF EXISTS update_current_delegations;
DROP TABLE IF EXISTS delegations_current;
DROP TABLE IF EXISTS delegations;

DROP TABLE IF EXISTS rewards;

DROP TABLE IF EXISTS network_limits;
DROP VIEW IF EXISTS orders_current;
DROP VIEW IF EXISTS orders_current_versions;

DROP VIEW IF EXISTS risk_factors_current;
drop table if exists risk_factors;
drop table if exists margin_levels cascade;
DROP TRIGGER IF EXISTS update_current_margin_levels ON margin_levels;
DROP FUNCTION IF EXISTS update_current_margin_levels;
DROP TABLE IF EXISTS current_margin_levels;

drop trigger if exists update_current_deposits on deposits;
drop function if exists update_current_deposits;
-- +goose StatementBegin
DO $$
    BEGIN
        IF EXISTS (SELECT relname
                   FROM pg_class
                   WHERE relname='deposits_current'
                     AND relkind = 'r')
        THEN
            DROP TABLE IF EXISTS deposits_current;
        ELSE
            DROP VIEW IF EXISTS deposits_current;
        END IF;
    END;
$$;
-- +goose StatementEnd
DROP TABLE IF EXISTS deposits cascade;
DROP TYPE IF EXISTS deposit_status;

drop trigger if exists update_current_withdrawals on withdrawals;
drop function if exists update_current_withdrawals;
-- +goose StatementBegin
DO $$
    BEGIN
        IF EXISTS (SELECT relname
                   FROM pg_class
                   WHERE relname='withdrawals_current'
                     AND relkind = 'r')
        THEN
            DROP TABLE IF EXISTS withdrawals_current;
        ELSE
            DROP VIEW IF EXISTS withdrawals_current;
        END IF;
    END;
$$;
-- +goose StatementEnd
DROP TABLE IF EXISTS withdrawals cascade;
DROP TYPE IF EXISTS withdrawal_status;


DROP TRIGGER IF EXISTS archive_orders ON orders;
DROP FUNCTION IF EXISTS archive_orders;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS orders_live;
DROP TABLE IF EXISTS orders_history;

DROP TYPE IF EXISTS order_time_in_force;
DROP TYPE IF EXISTS order_status;
DROP TYPE IF EXISTS order_side;
DROP TYPE IF EXISTS order_type;
DROP TYPE IF EXISTS order_pegged_reference;

DROP TABLE IF EXISTS ranking_scores;
DROP TABLE IF EXISTS reward_scores;
DROP TYPE IF EXISTS validator_node_status;

DROP TABLE IF EXISTS nodes;
DROP TABLE IF EXISTS nodes_announced;
DROP TYPE IF EXISTS node_status;

DROP TRIGGER IF EXISTS update_current_markets ON markets;
DROP FUNCTION IF EXISTS update_current_markets;
DROP TABLE IF EXISTS markets_current;
DROP TABLE IF EXISTS markets CASCADE;

DROP TABLE IF EXISTS markets;
DROP VIEW IF EXISTS market_data_snapshot;
DROP TABLE IF EXISTS market_data;
DROP TYPE IF EXISTS auction_trigger_type;
DROP TYPE IF EXISTS market_trading_mode_type;
DROP TYPE IF EXISTS market_state_type;

DROP TABLE IF EXISTS erc20_multisig_signer_events;
DROP TYPE IF EXISTS erc20_multisig_signer_event;

DROP TABLE IF EXISTS ledger;
DROP TABLE IF EXISTS balances cascade;
DROP TRIGGER IF EXISTS update_current_balances ON balances;
DROP FUNCTION IF EXISTS update_current_balances;
DROP TABLE IF EXISTS current_balances;

DROP TABLE IF EXISTS accounts;
DROP TABLE IF EXISTS parties;
DROP VIEW IF EXISTS assets_current;
DROP TABLE IF EXISTS assets;
DROP TYPE IF EXISTS asset_status_type;
DROP TABLE IF EXISTS trades cascade;
DROP TABLE IF EXISTS chain;
DROP TABLE IF EXISTS blocks cascade;
DROP TABLE IF EXISTS last_block cascade;
DROP TRIGGER IF EXISTS update_last_block ON balances;
DROP FUNCTION IF EXISTS update_last_block;


DROP DOMAIN IF EXISTS HUGEINT;
