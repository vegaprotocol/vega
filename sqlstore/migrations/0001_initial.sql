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
DROP AGGREGATE public.first(anyelement);
DROP AGGREGATE public.last(anyelement);
DROP FUNCTION public.first_agg(anyelement, anyelement);
DROP FUNCTION public.last_agg(anyelement, anyelement);

drop table ledger;
drop table balances;
drop table accounts;
drop table parties;
drop table assets;
drop table blocks;