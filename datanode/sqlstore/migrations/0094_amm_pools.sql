-- +goose Up

-- +goose StatementBegin
do $$
begin
    if not exists (select 1 from pg_type where typname = 'amm_pool_status') then
        create type amm_pool_status as enum(
            'STATUS_UNSPECIFIED', 'STATUS_ACTIVE', 'STATUS_REJECTED', 'STATUS_CANCELLED', 'STATUS_STOPPED'
        );
    end if;
end $$;
-- +goose StatementEnd

-- +goose StatementBegin
do $$
begin
    if not exists (select 1 from pg_type where typname = 'amm_pool_status_reason') then
        create type amm_pool_status_reason as enum(
            'STATUS_REASON_UNSPECIFIED', 'STATUS_REASON_CANCELLED_BY_PARTY', 'STATUS_REASON_CANNOT_FILL_COMMITMENT',
            'STATUS_REASON_PARTY_ALREADY_OWNS_A_POOL', 'STATUS_REASON_PARTY_CLOSED_OUT', 'STATUS_REASON_MARKET_CLOSED',
            'STATUS_REASON_COMMITMENT_TOO_LOW', 'STATUS_REASON_CANNOT_REBASE'
        );
    end if;
end $$;
-- +goose StatementEnd

create table if not exists amm_pool (
    party_id bytea not null,
    market_id bytea not null,
    pool_id bytea not null,
    sub_account bytea not null,
    commitment numeric not null,
    status amm_pool_status not null,
    status_reason amm_pool_status_reason not null,
    parameters_base numeric not null,
    parameters_lower_bound numeric not null,
    parameters_upper_bound numeric not null,
    parameters_margin_ratio_at_upper_bound numeric not null,
    parameters_margin_ratio_at_lower_bound numeric not null,
    created_at timestamp with time zone not null,
    last_updated timestamp with time zone not null,
    primary key (party_id, market_id, pool_id, sub_account)
);

-- +goose Down
drop table if exists amm_pool;
drop type if exists amm_pool_status_reason;
drop type if exists amm_pool_status;
