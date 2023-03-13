-- +goose Up

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION archive_orders()
   RETURNS TRIGGER
   LANGUAGE PLPGSQL AS
$$
BEGIN

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

drop index orders_id_current_idx;
drop index orders_market_id_vega_time_idx;
drop index orders_party_id_vega_time_idx;
drop index orders_reference_vega_time_idx;

drop view orders_current;
drop view orders_current_versions;

alter table orders drop column current;

CREATE VIEW orders_current_versions AS (
   SELECT DISTINCT ON (id, version) * FROM orders ORDER BY id, version DESC, vega_time DESC
);

alter table orders_live drop column current;

create index on orders (id, vega_time desc, seq_num desc);
create index on orders (created_at desc, id, vega_time desc, seq_num desc);
create index on orders (market_id, created_at desc, id, vega_time desc, seq_num desc);
create index on orders (party_id, created_at desc, id, vega_time desc, seq_num desc);
create index on orders (reference, created_at desc, id, vega_time desc, seq_num desc);

CREATE VIEW orders_current_desc
 AS
SELECT DISTINCT ON (orders.created_at, orders.id) *
FROM orders
ORDER BY orders.created_at DESC, orders.id, orders.vega_time DESC, orders.seq_num DESC;


CREATE VIEW orders_current_desc_by_market
 AS
SELECT DISTINCT ON (orders.created_at, orders.market_id, orders.id) *
FROM orders
ORDER BY orders.created_at DESC, orders.market_id, orders.id, orders.vega_time DESC, orders.seq_num DESC;

CREATE VIEW orders_current_desc_by_party
AS
SELECT DISTINCT ON (orders.created_at, orders.party_id, orders.id) *
        FROM orders
        ORDER BY orders.created_at DESC, orders.party_id, orders.id, orders.vega_time DESC, orders.seq_num DESC;

CREATE VIEW orders_current_desc_by_reference
AS
SELECT DISTINCT ON (orders.created_at, orders.reference, orders.id) *
        FROM orders
        ORDER BY orders.created_at DESC, orders.reference, orders.id, orders.vega_time DESC, orders.seq_num DESC;

-- Selecting current order by id to be done as follows -> select * from orders where id = <order id> order by vega_time desc, seq_num desc limit 1

-- +goose Down
drop view orders_current_desc;
drop view orders_current_desc_by_reference;
drop view orders_current_desc_by_party;
drop view orders_current_desc_by_market;


drop index orders_created_at_id_vega_time_seq_num_idx;
drop index orders_id_vega_time_seq_num_idx;
drop index orders_market_id_created_at_id_vega_time_seq_num_idx;
drop index orders_party_id_created_at_id_vega_time_seq_num_idx;
drop index orders_reference_created_at_id_vega_time_seq_num_idx;

alter table orders add column current BOOLEAN NOT NULL DEFAULT TRUE;
alter table orders_live add column  current BOOLEAN NOT NULL DEFAULT TRUE;

CREATE INDEX ON orders (market_id, vega_time DESC) where current=true;
CREATE INDEX ON orders (party_id, vega_time DESC) where current=true;
CREATE INDEX ON orders (reference, vega_time DESC) where current=true;
CREATE INDEX ON orders (id, current);

CREATE VIEW orders_current AS (
  SELECT * FROM orders WHERE current = true
  );

-- Restore the current flag

WITH current_orders AS (
    SELECT id, MAX(vega_time) AS max_vega_time
    FROM orders
    WHERE current = true
GROUP BY id
HAVING COUNT(id) > 1
    )
UPDATE orders o
SET current = CASE
                  WHEN o.vega_time = co.max_vega_time AND o.seq_num = (
                      SELECT MAX(seq_num)
                      FROM orders
                      WHERE id = o.id AND vega_time = co.max_vega_time
                  ) THEN true ELSE false
    END
    FROM current_orders co
WHERE o.id = co.id AND o.current = true;


-- Restore the original function that updates the current flag

-- +goose StatementBegin

CREATE OR REPLACE FUNCTION archive_orders()
   RETURNS TRIGGER
   LANGUAGE PLPGSQL AS
$$
BEGIN
    -- It is permitted by core to re-use order IDs and 'resurrect' done orders (specifically,
    -- LP orders do this, so we need to check our history table to see if we need to updated
    -- current flag on any most-recent-version-of an order.
UPDATE orders
SET current = false
WHERE current = true
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
              new.tx_hash, new.vega_time, new.seq_num, true);
END IF;

RETURN NEW;

END;
$$;

-- +goose StatementEnd