-- +goose Up

-- The following views were not regenerated when the stop orders tables were previously updated
-- as the views are used by the APIs some data could be missing or incorrect when returning data
-- from the stop orders table. This migration will regenerate the views to ensure the data is correct.
create or replace view stop_orders_current_desc
as
select distinct on (so.created_at, so.id) *
from stop_orders so
order by so.created_at desc, so.id, so.vega_time desc, so.seq_num desc;

create or replace view stop_orders_current_desc_by_market
as
select distinct on (so.created_at, so.market_id, so.id) *
from stop_orders so
order by so.created_at desc, so.market_id, so.id, so.vega_time desc, so.seq_num desc;

create or replace view stop_orders_current_desc_by_party
as
select distinct on (so.created_at, so.party_id, so.id) *
from stop_orders so
order by so.created_at desc, so.party_id, so.id, so.vega_time desc, so.seq_num desc;
