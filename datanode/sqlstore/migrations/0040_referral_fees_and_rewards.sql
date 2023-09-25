-- +goose Up

-- referral_fee_stats stores the per-epoch accumulated fees and rewards stats by asset
create table if not exists referral_fee_stats (
      market_id bytea not null,
      asset_id bytea not null,
      epoch_seq bigint not null,
      total_rewards_paid jsonb not null,
      referrer_rewards_generated jsonb not null,
      referees_discount_applied jsonb not null,
      volume_discount_applied jsonb not null,
      vega_time timestamp with time zone not null,
      primary key (vega_time, market_id, asset_id, epoch_seq)
);

select create_hypertable('referral_fee_stats', 'vega_time', chunk_time_interval => INTERVAL '1 day');

-- +goose Down

drop table if exists referral_fee_stats;
