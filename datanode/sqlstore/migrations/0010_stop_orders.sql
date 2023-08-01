-- +goose Up

create type stop_order_expiry_strategy as enum('EXPIRY_STRATEGY_UNSPECIFIED', 'EXPIRY_STRATEGY_CANCELS', 'EXPIRY_STRATEGY_SUBMIT');
create type stop_order_trigger_direction as enum('TRIGGER_DIRECTION_UNSPECIFIED', 'TRIGGER_DIRECTION_RISES_ABOVE', 'TRIGGER_DIRECTION_FALLS_BELOW');
create type stop_order_status as enum('STATUS_UNSPECIFIED', 'STATUS_PENDING', 'STATUS_CANCELLED', 'STATUS_STOPPED', 'STATUS_TRIGGERED', 'STATUS_EXPIRED', 'STATUS_REJECTED');

create table if not exists stop_orders(
    id bytea not null,
    oco_link_id bytea,
    expires_at timestamp with time zone,
    expiry_strategy stop_order_expiry_strategy not null,
    trigger_direction stop_order_trigger_direction not null,
    status stop_order_status not null,
    created_at timestamp with time zone not null,
    updated_at timestamp with time zone,
    order_id bytea not null,
    trigger_price text,
    trigger_percent_offset text,
    party_id bytea not null,
    market_id bytea not null,
    vega_time timestamp with time zone not null,
    seq_num bigint not null,
    tx_hash bytea not null,
    submission jsonb,
    primary key(vega_time, seq_num)
);

create index idx_stop_orders_id_vega_time_seq_num on stop_orders(id, vega_time desc, seq_num desc);
create index idx_stop_orders_created_at_id_vega_time_seq_num on stop_orders(created_at desc, id, vega_time desc, seq_num desc);
create index idx_stop_orders_market_id_created_at_id_vega_time_seq_num on stop_orders(market_id, created_at desc, id, vega_time desc, seq_num desc);
create index idx_stop_orders_party_id_created_at_id_vega_time_seq_num on stop_orders(party_id, created_at desc, id, vega_time desc, seq_num desc);

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

-- +goose Down

drop view if exists stop_orders_current_desc_by_party;
drop view if exists stop_orders_current_desc_by_market;
drop view if exists stop_orders_current_desc;
drop index if exists idx_stop_orders_party_id_created_at_id_vega_time_seq_num;
drop index if exists idx_stop_orders_market_id_created_at_id_vega_time_seq_num;
drop index if exists idx_stop_orders_created_at_id_vega_time_seq_num;
drop index if exists idx_stop_orders_id_vega_time_seq_num;
drop table if exists stop_orders;
drop type if exists stop_order_status;
drop type if exists stop_order_trigger_direction;
drop type if exists stop_order_expiry_strategy;
