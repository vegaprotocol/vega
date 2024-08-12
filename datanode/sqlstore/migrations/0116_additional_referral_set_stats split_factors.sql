-- +goose Up
alter table referral_set_stats ALTER COLUMN reward_factor DROP NOT NULL;
alter table referral_set_stats ALTER COLUMN rewards_factor_multiplier DROP NOT NULL;

alter table referral_set_stats
    add column reward_factors jsonb;

alter table referral_set_stats
    add column rewards_factors_multiplier jsonb;

update referral_set_stats set reward_factors = json_build_object('infrastructure_reward_factor', reward_factor, 'maker_reward_factor', reward_factor, 'liquidity_reward_factor', reward_factor);
update referral_set_stats set rewards_factors_multiplier = json_build_object('infrastructure_reward_factor', rewards_factor_multiplier, 'maker_reward_factor', rewards_factor_multiplier, 'liquidity_reward_factor', rewards_factor_multiplier);

UPDATE volume_discount_stats
SET parties_volume_discount_stats = (
  SELECT jsonb_agg(
    jsonb_set(
      party_stats,
      '{discount_factors}',
      jsonb_build_object(
        'infrastructure_discount_factor', party_stats->>'discount_factor',
        'liquidity_discount_factor', party_stats->>'discount_factor',
        'maker_discount_factor', party_stats->>'discount_factor'
      )
    )
  )
  FROM jsonb_array_elements(parties_volume_discount_stats) AS party_stats
);


alter table referral_set_stats DROP COLUMN reward_factor;
alter table referral_set_stats DROP COLUMN rewards_factor_multiplier;

alter table referral_set_stats ALTER COLUMN reward_factors SET NOT NULL;
alter table referral_set_stats ALTER COLUMN rewards_factors_multiplier SET NOT NULL;

alter table trades ADD COLUMN buyer_buy_back_fee HUGEINT NOT NULL DEFAULT(0),
                   ADD COLUMN buyer_treasury_fee HUGEINT NOT NULL DEFAULT(0),
                   ADD COLUMN buyer_high_volume_maker_fee HUGEINT NOT NULL DEFAULT(0),
                   ADD COLUMN seller_buy_back_fee HUGEINT NOT NULL DEFAULT(0),
                   ADD COLUMN seller_treasury_fee HUGEINT NOT NULL DEFAULT(0),
                   ADD COLUMN seller_high_volume_maker_fee HUGEINT NOT NULL DEFAULT(0);

-- create a new volume rebate record when a new volume rebate program is created,
-- updated, or ended so that we keep an audit trail, just in case.
-- We create it as a hypertable and set a retention policy to make sure
-- old and redundant data is removed in due course.
create table if not exists volume_rebate_programs
(
    id                       bytea                    not null,
    version                  int                      not null,
    benefit_tiers            jsonb,
    end_of_program_timestamp timestamp with time zone not null,
    window_length            int                      not null,
    vega_time                timestamp with time zone not null,
    ended_at                 timestamp with time zone,
    seq_num                  bigint                   not null,
    primary key (vega_time, seq_num)
);

select create_hypertable('volume_rebate_programs', 'vega_time', chunk_time_interval => INTERVAL '1 day');

-- simplify volume rebate retrieval using a view that provides the latest volume rebate information.
create view current_volume_rebate_program as
(
select *
from volume_rebate_programs
order by vega_time desc, seq_num desc
limit 1 -- there should only be 1 volume rebate program running at any time, so just get the last record.
    );

create table volume_rebate_stats
(
    at_epoch                      bigint                   not null,
    parties_volume_rebate_stats jsonb                    not null,
    vega_time                     timestamp with time zone not null,
    primary key (at_epoch, vega_time)
);

select create_hypertable('volume_rebate_stats', 'vega_time', chunk_time_interval => INTERVAL '1 day');

ALTER TYPE proposal_error ADD VALUE IF NOT EXISTS 'PROPOSAL_ERROR_INVALID_VOLUME_REBATE_PROGRAM';

-- +goose Down

drop table if exists volume_rebate_stats;
drop view if exists current_volume_rebate_program;
drop table if exists volume_rebate_programs;

alter table referral_set_stats
    add column reward_factor text not null default '0';

alter table referral_set_stats
    add column rewards_factor_multiplier text not null default '0';

update referral_set_stats set reward_factor = 'reward_factor->infrastructure_reward_factor';
update referral_set_stats set rewards_factor_multiplier = 'rewards_factors_multiplier->infrastructure_reward_factor';

alter table referral_set_stats DROP COLUMN reward_factors;
alter table referral_set_stats DROP COLUMN rewards_factors_multiplier;

alter table trades DROP COLUMN buyer_buy_back_fee,
                   DROP COLUMN buyer_treasury_fee,
                   DROP COLUMN seller_buy_back_fee,
                   DROP COLUMN seller_treasury_fee,
                   DROP COLUMN buyer_high_volume_maker_fee,
                   DROP COLUMN seller_high_volume_maker_fee;
