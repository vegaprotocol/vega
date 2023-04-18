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
-- As per https://github.com/vegaprotocol/vega/issues/8149, only LIMIT type (1) orders with status active (1) and parked (8)
-- and time_in_force != IOC (3) and time_in_force != FOK (4) are considered live.
IF NEW.status IN (1, 8) AND NEW.type = 1 AND NEW.time_in_force NOT IN (3, 4)
    THEN
       INSERT INTO orders_live
       VALUES(new.id, new.market_id, new.party_id, new.side, new.price,
              new.size, new.remaining, new.time_in_force, new.type, new.status,
              new.reference, new.reason, new.version, new.batch_id, new.pegged_offset,
              new.pegged_reference, new.lp_id, new.created_at, new.updated_at, new.expires_at,
              new.tx_hash, new.vega_time, new.seq_num, new.post_only, new.reduce_only);
END IF;

RETURN NEW;

END;
$$;

-- +goose StatementEnd

-- +goose Down

-- revert it back to what it was before.
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
