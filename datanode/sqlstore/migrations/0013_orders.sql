
-- +goose Up

ALTER TABLE orders rename to orders_old;

CREATE TABLE orders_initial (
    id                BYTEA                     NOT NULL,
    market_id         BYTEA                     NOT NULL,
    party_id          BYTEA                     NOT NULL,
    side              SMALLINT                  NOT NULL,
    reference         TEXT,
    lp_id             BYTEA,
    created_at        TIMESTAMP WITH TIME ZONE NOT NULL,
    post_only         BOOLEAN DEFAULT false,
    reduce_only       BOOLEAN DEFAULT false,
    tx_hash           BYTEA                    NOT NULL,
    vega_time         TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY(created_at, id) -- Actually unique on ID but need vega_time in there to make a hypertable
);

CREATE INDEX ON orders_initial(id);
CREATE INDEX ON orders_initial(market_id, created_at DESC); -- For querying without any filter
CREATE INDEX ON orders_initial(party_id, created_at DESC); -- For querying without any filter
CREATE INDEX ON orders_initial(reference, created_at DESC); -- For querying by reference

SELECT create_hypertable('orders_initial', 'created_at', chunk_time_interval => INTERVAL '1 day');

---------------- Order Updates

CREATE TABLE order_updates (
    id                BYTEA                     NOT NULL,
    price             HUGEINT                    NOT NULL,
    size              BIGINT                    NOT NULL,
    remaining         BIGINT                    NOT NULL,
    time_in_force     SMALLINT                  NOT NULL,
    type              SMALLINT                  NOT NULL,
    status            SMALLINT                  NOT NULL,
    reason            SMALLINT,
    version           INT                       NOT NULL,
    batch_id          INT                       NOT NULL,
    pegged_offset     HUGEINT,
    pegged_reference  SMALLINT,
    created_at        TIMESTAMP WITH TIME ZONE,
    updated_at        TIMESTAMP WITH TIME ZONE,
    expires_at        TIMESTAMP WITH TIME ZONE,
    seq_num           BIGINT                   NOT NULL,
    tx_hash           BYTEA                    NOT NULL,
    vega_time         TIMESTAMP WITH TIME ZONE NOT NULL,
    PRIMARY KEY(vega_time, seq_num)
);
CREATE INDEX ON order_updates(id, vega_time desc, seq_num desc);

SELECT create_hypertable('order_updates', 'vega_time', chunk_time_interval => INTERVAL '1 day');

---------------- Job to delete rows from orders_inital once retention deltes rows from order_udpates

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION prune_orders_initial()
RETURNS VOID AS $$
BEGIN
    DELETE FROM orders_initial oi
    WHERE NOT EXISTS (
        SELECT 1
        FROM order_updates ou
        WHERE ou.id = oi.id
        );
END
$$ LANGUAGE plpgsql;

SELECT add_job('prune_orders_initial', INTERVAL '1 day');
-- +goose StatementEnd

---------------- Some nice views

CREATE OR REPLACE VIEW orders_current as (
    select
        initial.id as id,
        initial.market_id,
        initial.party_id,
        initial.side,
        initial.reference,
        initial.lp_id,
        initial.created_at,
        initial.tx_hash as tx_hash_initial,
        initial.vega_time as vega_time_initial,
        initial.post_only,
        initial.reduce_only,
        latest.price,
        latest.size,
        latest.remaining,
        latest.time_in_force,
        latest.type,
        latest.status,
        latest.reason,
        latest.version,
        latest.batch_id,
        latest.pegged_offset,
        latest.pegged_reference,
        latest.updated_at,
        latest.expires_at,
        latest.seq_num,
        latest.tx_hash,
        latest.vega_time
    from orders_initial as initial
    left join lateral (select * from order_updates where id=initial.id order by id, vega_time desc, seq_num desc limit 1) latest ON true
    order by created_at desc, id
);

CREATE OR REPLACE VIEW orders as (
    SELECT
        orders_initial.id as id,
        orders_initial.market_id,
        orders_initial.party_id,
        orders_initial.side,
        orders_initial.reference,
        orders_initial.lp_id,
        orders_initial.created_at,
        orders_initial.post_only,
        orders_initial.reduce_only,        
        order_updates.price,
        order_updates.size,
        order_updates.remaining,
        order_updates.time_in_force,
        order_updates.type,
        order_updates.status,
        order_updates.reason,
        order_updates.version,
        order_updates.batch_id,
        order_updates.pegged_offset,
        order_updates.pegged_reference,
        order_updates.updated_at,
        order_updates.expires_at,
        order_updates.seq_num,
        order_updates.tx_hash,
        order_updates.vega_time
    FROM order_updates INNER JOIN orders_initial ON order_updates.id = orders_initial.ID
);

DROP VIEW orders_current_versions;
CREATE VIEW orders_current_versions AS (
   SELECT DISTINCT ON (id, version) * FROM orders ORDER BY id, version DESC, vega_time DESC
);

---------------- When inserting into orders view, split out into orders_inital, order_updates and orders_live

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION order_insert_func()
   RETURNS TRIGGER
   LANGUAGE PLPGSQL AS
$$
BEGIN

INSERT INTO orders_initial
    VALUES(NEW.id, NEW.market_id, NEW.party_id, NEW.side, NEW.reference, NEW.lp_id,
           NEW.created_at, NEW.post_only, NEW.reduce_only, NEW.tx_hash, NEW.vega_time)
    ON CONFLICT (created_at, id) DO NOTHING;

INSERT INTO order_updates
    VALUES( NEW.id, NEW.price, NEW.size, NEW.remaining, NEW.time_in_force,
            NEW.type, NEW.status, NEW.reason, NEW.version, NEW.batch_id,
            NEW.pegged_offset, NEW.pegged_reference, NEW.created_at,  NEW.updated_at, NEW.expires_at,
            NEW.seq_num, NEW.tx_hash, NEW.vega_time);

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
              new.tx_hash, new.vega_time, new.seq_num);
END IF;

RETURN NEW;

END;
$$;

-- +goose StatementEnd

CREATE TRIGGER order_insert_trigger INSTEAD of INSERT ON orders for each ROW EXECUTE PROCEDURE order_insert_func();


---------------- Migrate Data

INSERT INTO orders_initial(
    id,
    market_id,
    party_id,
    side,
    reference,
    lp_id,
    created_at,
    post_only,
    reduce_only,
    tx_hash,
    vega_time)
    (SELECT DISTINCT ON (id)
        id,
        market_id,
        party_id,
        side,
        reference,
        lp_id,
        created_at,
        post_only,
        reduce_only,        
        tx_hash,
        vega_time
      FROM orders_old ORDER BY id, vega_time DESC, seq_num desc);


INSERT INTO order_updates(
    id,
    price,
    size,
    remaining,
    time_in_force,
    type,
    status,
    reason,
    version,
    batch_id,
    pegged_offset,
    pegged_reference,
    created_at, -- duplicated to help with retention
    updated_at,
    expires_at,
    seq_num,
    tx_hash,
    vega_time
)
SELECT id,
    price,
    size,
    remaining,
    time_in_force,
    type,
    status,
    reason,
    version,
    batch_id,
    pegged_offset,
    pegged_reference,
    created_at,
    updated_at,
    expires_at,
    seq_num,
    tx_hash,
    vega_time
FROM orders_old;


DROP VIEW orders_current_desc;
DROP VIEW orders_current_desc_by_market;
DROP VIEW orders_current_desc_by_party;
DROP VIEW orders_current_desc_by_reference;
DROP TABLE orders_old;

-- +goose Down

DROP VIEW orders_current_versions;
DROP VIEW orders;   
DROP VIEW orders_current;
DROP TABLE order_updates;
DROP TABLE orders_initial;

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
    post_only         BOOLEAN DEFAULT false,
    reduce_only       BOOLEAN DEFAULT false,
    PRIMARY KEY(vega_time, seq_num)
);


create index on orders (id, vega_time desc, seq_num desc);
create index on orders (created_at desc, id, vega_time desc, seq_num desc);
create index on orders (market_id, created_at desc, id, vega_time desc, seq_num desc);
create index on orders (party_id, created_at desc, id, vega_time desc, seq_num desc);
create index on orders (reference, created_at desc, id, vega_time desc, seq_num desc);

CREATE OR REPLACE VIEW orders_current_versions AS (
   SELECT DISTINCT ON (id, version) * FROM orders ORDER BY id, version DESC, vega_time DESC
);

CREATE OR REPLACE VIEW orders_current_desc
 AS
SELECT DISTINCT ON (orders.created_at, orders.id) *
FROM orders
ORDER BY orders.created_at DESC, orders.id, orders.vega_time DESC, orders.seq_num DESC;


CREATE OR REPLACE VIEW orders_current_desc_by_market
 AS
SELECT DISTINCT ON (orders.created_at, orders.market_id, orders.id) *
FROM orders
ORDER BY orders.created_at DESC, orders.market_id, orders.id, orders.vega_time DESC, orders.seq_num DESC;

CREATE OR REPLACE VIEW orders_current_desc_by_party
AS
SELECT DISTINCT ON (orders.created_at, orders.party_id, orders.id) *
        FROM orders
        ORDER BY orders.created_at DESC, orders.party_id, orders.id, orders.vega_time DESC, orders.seq_num DESC;

CREATE OR REPLACE VIEW orders_current_desc_by_reference
AS
SELECT DISTINCT ON (orders.created_at, orders.reference, orders.id) *
        FROM orders
        ORDER BY orders.created_at DESC, orders.reference, orders.id, orders.vega_time DESC, orders.seq_num DESC;

