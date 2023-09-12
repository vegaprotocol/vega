-- +goose Up

-- create a new referral record when a new referral program is created,
-- updated, or ended so that we keep an audit trail, just in case.
-- We create it as a hypertable and set a retention policy to make sure
-- old and redundant data is removed in due course.
create table if not exists referral_programs (
  id bytea not null,
  version int not null,
  benefit_tiers jsonb,
  end_of_program_timestamp timestamp with time zone not null,
  window_length int not null,
  staking_tiers jsonb,
  vega_time timestamp with time zone not null,
  ended_at timestamp with time zone,
  primary key (vega_time)
);

select create_hypertable('referral_programs', 'vega_time', chunk_time_interval => INTERVAL '1 day');

-- simplify referral retrieval using a view that provides the latest referral information.
create view current_referral_program as (
    select *
    from referral_programs
    order by vega_time desc limit 1 -- there should only be 1 referral program running at any time, so just get the last record.
);

create table referral_sets(
    id bytea not null,
    referrer bytea not null,
    created_at timestamp with time zone not null,
    updated_at timestamp with time zone not null,
    vega_time timestamp with time zone not null,
    primary key (id)
);

create table referral_set_referees(
    referral_set_id bytea not null,
    referee bytea not null,
    joined_at timestamp with time zone not null,
    at_epoch bigint not null,
    vega_time timestamp with time zone not null,
    primary key (referral_set_id, referee)
);

create table referral_set_stats(
    set_id bytea not null,
    at_epoch bigint not null,
    referral_set_running_notional_taker_volume numeric,
    referees_stats jsonb not null,
    vega_time timestamp with time zone not null,
    primary key (vega_time, set_id)
);

select create_hypertable('referral_set_stats', 'vega_time', chunk_time_interval => INTERVAL '1 day');

-- make sure that it doesn't exist in case migrations have failed previously
drop view if exists referral_set_referee_stats;

create view referral_set_referee_stats as (
  select set_id, at_epoch, referral_set_running_notional_taker_volume, stats.referee_stats->>'party_id' as party_id,
         stats.referee_stats->>'discount_factor' as discount_factor, stats.referee_stats->>'reward_factor' as reward_factor,
         vega_time
  from referral_set_stats,
       jsonb_array_elements(referees_stats) with ordinality stats(referee_stats, position)
);

-- +goose Down

drop view if exists referral_set_referee_stats;
drop table if exists referral_set_stats;
drop table if exists referral_set_referees;
drop table if exists referral_sets;
drop view if exists current_referral_program;
drop table if exists referral_programs;
