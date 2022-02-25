-- +goose Up
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

-- +goose Down
DROP AGGREGATE IF EXISTS public.first(anyelement);
DROP AGGREGATE IF EXISTS public.last(anyelement);
DROP FUNCTION IF EXISTS public.first_agg(anyelement, anyelement);
DROP FUNCTION IF EXISTS public.last_agg(anyelement, anyelement);

DROP TABLE IF EXISTS network_limits;
DROP VIEW IF EXISTS orders_current;
DROP VIEW IF EXISTS orders_current_versions;

DROP TABLE IF EXISTS orders;
DROP TYPE IF EXISTS order_time_in_force;
DROP TYPE IF EXISTS order_status;
DROP TYPE IF EXISTS order_side;
DROP TYPE IF EXISTS order_type;
DROP TYPE IF EXISTS order_pegged_reference;

DROP TABLE IF EXISTS ledger;
DROP TABLE IF EXISTS balances;
DROP TABLE IF EXISTS accounts;
DROP TABLE IF EXISTS parties;
DROP TABLE IF EXISTS assets;
DROP TABLE IF EXISTS blocks;