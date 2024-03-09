-- +goose Up

create table time_weighted_notional_positions (
    asset_id bytea not null,
    party_id bytea not null,
    epoch_seq bigint not null,
    time_weighted_notional_position hugeint not null,
    last_updated timestamp with time zone not null,
    primary key (asset_id, party_id, epoch_seq)
);

-- +goose Down

drop table if exists time_weighted_notional_positions;
