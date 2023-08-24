-- +goose Up

-- +goose StatementBegin
DO $$
BEGIN
    IF NOT EXISTS (SELECT * FROM timescaledb_information.hypertables WHERE hypertable_name = 'stop_orders') THEN
        PERFORM create_hypertable('stop_orders', 'vega_time', chunk_time_interval => INTERVAL '1 day');
    END IF;
END $$;
-- +goose StatementEnd

create table if not exists stop_orders_live (
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
    primary key(id)
);

create index idx_stop_orders_live_id_vega_time_seq_num on stop_orders_live(id, vega_time desc, seq_num desc);
create index idx_stop_orders_live_created_at_id_vega_time_seq_num on stop_orders_live(created_at desc, id, vega_time desc, seq_num desc);
create index idx_stop_orders_live_market_id_created_at_id_vega_time_seq_num on stop_orders_live(market_id, created_at desc, id, vega_time desc, seq_num desc);
create index idx_stop_orders_live_party_id_created_at_id_vega_time_seq_num on stop_orders_live(party_id, created_at desc, id, vega_time desc, seq_num desc);

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
                new.market_id, new.vega_time, new.seq_num, new.tx_hash, new.submission);
    end if;

    return new;
end;
$$;
-- +goose StatementEnd

create trigger stop_orders_live_insert_trigger before insert on stop_orders for each row execute function stop_orders_live_insert_trigger();

-- +goose Down

drop trigger if exists stop_orders_live_insert_trigger on stop_orders;
drop function if exists stop_orders_live_insert_trigger;
drop table if exists stop_orders_live;
