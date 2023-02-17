-- +goose Up

-- CREATE TABLE orders_new AS orders;
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_pkey;

-- ALTER TABLE orders ADD CONSTRAINT orders_pkey PRIMARY KEY (vega_time, seq_num, id);

-- +goose Down

-- ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_pkey;

ALTER TABLE orders ADD CONSTRAINT orders_pkey PRIMARY KEY (vega_time, seq_num); -- not sure if we really can go back
