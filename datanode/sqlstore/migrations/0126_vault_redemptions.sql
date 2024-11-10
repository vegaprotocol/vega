-- +goose Up

CREATE TYPE redeem_status AS enum('REDEEM_STATUS_UNSPECIFIED','REDEEM_STATUS_PENDING','REDEEM_STATUS_LATE','REDEEM_STATUS_COMPLETED');

-- create the history and current tables for vault state
create table if not exists vault_redemption_request (
       request_id bytea not null,
       vault_id bytea not null,
       party_id bytea not null,
       asset bytea not null,
       requested_amount NUMERIC,
       remaining_amount NUMERIC,
       eligibility_date timestamp with time zone not null,
       last_updated timestamp with time zone not null,
       status redeem_status,
       vega_time timestamp with time zone not null,
       primary key (vega_time, request_id)
);

select create_hypertable('vault_redemption_request', 'vega_time', chunk_time_interval => INTERVAL '1 day');

create table if not exists vault_redemption_request_current (
       request_id bytea not null,
       vault_id bytea not null,
       party_id bytea not null,
       asset bytea not null,
       requested_amount NUMERIC,
       remaining_amount NUMERIC,
       eligibility_date timestamp with time zone not null,
       last_updated timestamp with time zone not null,
       status redeem_status,
       vega_time timestamp with time zone not null,
       primary key (request_id)
);

-- create the trigger functions and triggers
-- +goose StatementBegin
create or replace function update_vault_redemption_request()
       returns trigger
       language plpgsql
as $$
   begin
        insert into vault_redemption_request_current(request_id, vault_id, party_id, asset, requested_amount, remaining_amount, eligibility_date, last_updated, status, vega_time)
        values (new.request_id, new.vault_id, new.party_id, new.asset, new.requested_amount, new.remaining_amount, new.eligibility_date, new.last_updated, new.status, new.vega_time)
        on conflict(request_id)
        do update set
           requested_amount = excluded.requested_amount,
           remaining_amount = excluded.remaining_amount,
           status = excluded.status,
           eligibility_date = excluded.eligibility_date,
           last_updated = excluded.last_updated,
           vega_time = excluded.vega_time;
        return null;
   end;
$$;

-- +goose StatementEnd

create trigger update_vault_redemption_request
    after insert or update
    on vault_redemption_request
    for each row execute function update_vault_redemption_request();

-- +goose Down

drop table if exists vault_redemption_request_current;
drop trigger if exists update_vault_redemption_request on vault_redemption_request;
drop function if exists update_vault_redemption_request;
drop table if exists vault_redemption_request;

drop type if exists redeem_status;
