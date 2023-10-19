-- +goose Up

alter table fees_stats
    add column total_maker_fees_received jsonb not null default '[]';

alter table fees_stats
    add column maker_fees_generated jsonb not null default '[]';

alter table fees_stats
    rename column total_rewards_paid TO total_rewards_received;

create table if not exists fees_stats_by_party
(
    market_id                 bytea                    not null,
    asset_id                  bytea                    not null,
    party_id                  bytea                    not null,
    epoch_seq                 bigint                   not null,
    total_rewards_received    hugeint                  not null,
    referees_discount_applied hugeint                  not null,
    volume_discount_applied   hugeint                  not null,
    total_maker_fees_received hugeint                  not null,
    vega_time                 timestamp with time zone not null,
    PRIMARY KEY (party_id, market_id, asset_id, vega_time)
);

create index fees_stats_by_party_market_party on fees_stats_by_party
    using btree (party_id, asset_id);

-- +goose Down

alter table fees_stats
    drop column total_maker_fees_received;

alter table fees_stats
    drop column maker_fees_generated;

alter table fees_stats
    rename column total_rewards_received TO total_rewards_paid;
