-- +goose Up

ALTER TABLE orders
      RENAME COLUMN initial_peak_size TO peak_size;
ALTER TABLE orders
      RENAME COLUMN minimum_peak_size TO minimum_visible_size;
ALTER TABLE orders_live
      RENAME COLUMN initial_peak_size TO peak_size;
ALTER TABLE orders_live
      RENAME COLUMN minimum_peak_size TO minimum_visible_size;

drop view orders_current_versions;
drop view orders_current_desc;
drop view orders_current_desc_by_reference;
drop view orders_current_desc_by_party;
drop view orders_current_desc_by_market;

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
      RENAME COLUMN peak_size TO initial_peak_size;
ALTER TABLE orders
      RENAME COLUMN minimum_visible_size TO minimum_peak_size;
ALTER TABLE orders_live
      RENAME COLUMN peak_size TO initial_peak_size;
ALTER TABLE orders_live
      RENAME COLUMN minimum_visible_size TO minimum_peak_size;

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