-- +goose Up

ALTER TABLE orders
      ADD COLUMN reserved_remaining BIGINT,
      ADD COLUMN initial_peak_size BIGINT,
      ADD COLUMN minimum_peak_size BIGINT;

ALTER TABLE orders_live
      ADD COLUMN reserved_remaining BIGINT,
      ADD COLUMN initial_peak_size BIGINT,
      ADD COLUMN minimum_peak_size BIGINT;

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


-- +goose Down

drop view orders_current_versions;
drop view orders_current_desc;
drop view orders_current_desc_by_reference;
drop view orders_current_desc_by_party;
drop view orders_current_desc_by_market;


ALTER TABLE orders
      DROP COLUMN IF EXISTS reserved_remaining,
      DROP COLUMN IF EXISTS initial_peak_size,
      DROP COLUMN IF EXISTS minimum_peak_size;

ALTER TABLE orders_live
      DROP COLUMN IF EXISTS reserved_remaining,
      DROP COLUMN IF EXISTS initial_peak_size,
      DROP COLUMN IF EXISTS minimum_peak_size;

CREATE OR REPLACE VIEW orders_current_versions AS (
   SELECT DISTINCT ON (id, version) * FROM orders ORDER BY id, version DESC, vega_time DESC
);

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