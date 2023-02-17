-- +goose Up

ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_pkey;

ALTER TABLE orders ADD CONSTRAINT PRIMARY KEY (vega_time, seq_num, id);

-- +goose Down

ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_pkey;

ALTER TABLE orders ADD CONSTRAINT PRIMARY KEY (vega_time, seq_num); -- not sure if we really can go back
