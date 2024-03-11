-- +goose Up

create table time_weighted_notional_positions (
    asset_id bytea not null,
    party_id bytea not null,
    epoch_seq bigint not null,
    time_weighted_notional_position hugeint not null,
    vega_time timestamp with time zone not null,
    primary key (asset_id, party_id, epoch_seq, vega_time)
);

select create_hypertable('time_weighted_notional_positions', 'vega_time', chunk_time_interval => interval '1 day', if_not_exists => true);

-- +goose Down

drop table if exists time_weighted_notional_positions;
