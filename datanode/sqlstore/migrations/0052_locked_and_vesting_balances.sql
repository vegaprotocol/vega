-- +goose Up

-- create the history and current tables for locked and vesting balances
create table if not exists party_locked_balances (
       party_id bytea not null,
       asset_id bytea not null,
       at_epoch bigint not null,
       until_epoch bigint not null,
       balance hugeint not null,
       vega_time timestamp with time zone not null,
       primary key (vega_time, party_id, asset_id)
);

select create_hypertable('party_locked_balances', 'vega_time', chunk_time_interval => INTERVAL '1 day');

create table if not exists party_locked_balances_current (
       party_id bytea not null,
       asset_id bytea not null,
       at_epoch bigint not null,
       until_epoch bigint not null,
       balance hugeint not null,
       vega_time timestamp with time zone not null,
       primary key (party_id, asset_id)
);

create table if not exists party_vesting_balances (
       party_id bytea not null,
       asset_id bytea not null,
       at_epoch bigint not null,
       balance hugeint not null,
       vega_time timestamp with time zone not null,
       primary key (vega_time, party_id, asset_id)
);

select create_hypertable('party_vesting_balances', 'vega_time', chunk_time_interval => INTERVAL '1 day');

create table if not exists party_vesting_balances_current (
       party_id bytea not null,
       asset_id bytea not null,
       at_epoch bigint not null,
       balance hugeint not null,
       vega_time timestamp with time zone not null,
       primary key (party_id, asset_id)
);

-- create the trigger functions and triggers
-- +goose StatementBegin
create or replace function update_party_locked_balances()
       returns trigger
       language plpgsql
as $$
   begin
        insert into party_locked_balances_current(party_id, asset_id, at_epoch, until_epoch, balance, vega_time)
        values (new.party_id, new.asset_id, new.at_epoch, new.until_epoch, new.balance, new.vega_time)
        on conflict(party_id, asset_id)
        do update set
           at_epoch = excluded.at_epoch,
           until_epoch = excluded.until_epoch,
           balance = excluded.balance,
           vega_time = excluded.vega_time;
        return null;
   end;
$$;
-- +goose StatementEnd

create trigger update_party_locked_balances
    after insert or update
    on party_locked_balances
    for each row execute function update_party_locked_balances();

-- +goose StatementBegin
create or replace function update_party_vesting_balances()
       returns trigger
       language plpgsql
as $$
begin
insert into party_vesting_balances_current(party_id, asset_id, at_epoch, balance, vega_time)
values (new.party_id, new.asset_id, new.at_epoch, new.balance, new.vega_time)
    on conflict(party_id, asset_id)
        do update set
            at_epoch = excluded.at_epoch,
            balance = excluded.balance,
            vega_time = excluded.vega_time;
    return null;
end;
$$;
-- +goose StatementEnd

create trigger update_party_vesting_balances
    after insert or update
    on party_vesting_balances
    for each row execute function update_party_vesting_balances();

-- +goose Down

drop table if exists party_vesting_balances_current;
drop table if exists party_vesting_balances;
drop table if exists party_locked_balances_current;
drop table if exists party_locked_balances;

drop function if exists update_party_locked_balances;
drop function if exists update_party_vesting_balances;
