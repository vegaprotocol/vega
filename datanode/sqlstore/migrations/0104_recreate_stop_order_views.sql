-- +goose Up

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

alter table stop_orders_live
    add column if not exists size_override_setting int not null default 0,
    add column if not exists size_override_value varchar null;

-- +goose StatementBegin
create or replace function stop_orders_live_insert_trigger()
returns trigger
    language plpgsql
    as $$
begin
    delete from stop_orders_live
    where id = new.id;

    if new.status in ('STATUS_UNSPECIFIED', 'STATUS_PENDING') then
        insert into stop_orders_live
        values (new.id, new.oco_link_id, new.expires_at, new.expiry_strategy, new.trigger_direction, new.status,
                new.created_at, new.updated_at, new.order_id, new.trigger_price, new.trigger_percent_offset, new.party_id,
                new.market_id, new.vega_time, new.seq_num, new.tx_hash, new.submission, new.size_override_setting, new.size_override_value);
    end if;

    return new;
end;
$$;

-- +goose StatementEnd

drop trigger if exists stop_orders_live_insert_trigger on stop_orders;
create trigger stop_orders_live_insert_trigger before insert on stop_orders for each row execute function stop_orders_live_insert_trigger();

-- +goose Down

-- restore the views to how they were befpre the migration or network history may fail due to migrations failing
drop view if exists stop_orders_current_desc;
create view stop_orders_current_desc
as
select distinct on (so.created_at, so.id) id, oco_link_id, expires_at, expiry_strategy, trigger_direction, status, created_at,
        updated_at, order_id, trigger_price, trigger_percent_offset, party_id, market_id, vega_time, seq_num, tx_hash, submission
        from stop_orders so
        order by so.created_at desc, so.id, so.vega_time desc, so.seq_num desc;

drop view if exists stop_orders_current_desc_by_market;
create view stop_orders_current_desc_by_market
as
select distinct on (so.created_at, so.market_id, so.id)id, oco_link_id, expires_at, expiry_strategy, trigger_direction, status, created_at,
        updated_at, order_id, trigger_price, trigger_percent_offset, party_id, market_id, vega_time, seq_num, tx_hash, submission
        from stop_orders so
        order by so.created_at desc, so.market_id, so.id, so.vega_time desc, so.seq_num desc;

drop view if exists stop_orders_current_desc_by_party;
create view stop_orders_current_desc_by_party
as
select distinct on (so.created_at, so.party_id, so.id)id, oco_link_id, expires_at, expiry_strategy, trigger_direction, status, created_at,
        updated_at, order_id, trigger_price, trigger_percent_offset, party_id, market_id, vega_time, seq_num, tx_hash, submission
        from stop_orders so
        order by so.created_at desc, so.party_id, so.id, so.vega_time desc, so.seq_num desc;
