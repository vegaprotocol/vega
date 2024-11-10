-- +goose Up

CREATE TYPE vault_status AS enum('VAULT_STATUS_UNSPECIFIED','VAULT_STATUS_ACTIVE','VAULT_STATUS_STOPPING','VAULT_STATUS_STOPPED');

-- create the history and current tables for vault state
create table if not exists vault_state (
       vault_id bytea not null,
       vault JSONB          NOT NULL,
       invested_amount NUMERIC,
       status vault_status,
       next_fee_calc timestamp with time zone not null,
       next_redemption_date timestamp with time zone not null,
       vega_time timestamp with time zone not null,
       primary key (vega_time, vault_id)
);

select create_hypertable('vault_state', 'vega_time', chunk_time_interval => INTERVAL '1 day');

create table if not exists vault_state_current (
       vault_id bytea not null,
       vault JSONB          NOT NULL,
       invested_amount NUMERIC,
       status vault_status,
       next_fee_calc timestamp with time zone not null,
       next_redemption_date timestamp with time zone not null,
       vega_time timestamp with time zone not null,
       primary key (vault_id)
);


-- create the history and current tables for vault state
create table if not exists vault_party_shares (
       vault_id bytea not null,
       party_id bytea not null,
       share NUMERIC,
       vega_time timestamp with time zone not null,
       primary key (vega_time, vault_id, party_id)
);

select create_hypertable('vault_party_shares', 'vega_time', chunk_time_interval => INTERVAL '1 day');

create table if not exists vault_party_shares_current (
       vault_id bytea not null,
       party_id bytea not null,
       share NUMERIC,
       vega_time timestamp with time zone not null,
       primary key (vault_id, party_id)
);

-- create the trigger functions and triggers
-- +goose StatementBegin
create or replace function update_vault_state()
       returns trigger
       language plpgsql
as $$
   begin
        insert into vault_state_current(vault_id, vault, invested_amount, status, next_fee_calc, next_redemption_date, vega_time)
        values (new.vault_id, new.vault, new.invested_amount, new.status, new.next_fee_calc, new.next_redemption_date, new.vega_time)
        on conflict(vault_id)
        do update set
           vault = excluded.vault,
           invested_amount = excluded.invested_amount,
           status = excluded.status,
           next_fee_calc = excluded.next_fee_calc,
           next_redemption_date = excluded.next_redemption_date,
           vega_time = excluded.vega_time;
        return null;
   end;
$$;

create or replace function update_vault_party_shares()
       returns trigger
       language plpgsql
as $$
   begin
        insert into vault_party_shares_current(vault_id, party_id, share, vega_time)
        values (new.vault_id, new.party_id, new.share, new.vega_time)
        on conflict(vault_id, party_id)
        do update set
           share = excluded.share,           
           vega_time = excluded.vega_time;
        return null;
   end;
$$;
-- +goose StatementEnd

create trigger update_vault_state
    after insert or update
    on vault_state
    for each row execute function update_vault_state();

create trigger update_vault_party_shares_state
    after insert or update
    on vault_party_shares
    for each row execute function update_vault_party_shares();

-- +goose Down

drop table if exists vault_state_current;
drop table if exists vault_state;
drop function if exists update_vault_state;

drop table if exists vault_party_shares_current;
drop table if exists vault_party_shares;
drop function if exists update_vault_party_shares;

drop type if exists vault_status;