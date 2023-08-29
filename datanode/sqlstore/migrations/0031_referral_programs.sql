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

-- +goose Down

drop view if exists current_referral_program;
drop table if exists referral_programs;
