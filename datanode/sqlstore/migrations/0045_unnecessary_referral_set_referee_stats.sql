-- +goose Up

drop view if exists referral_set_referee_stats;

alter table referral_set_stats
    alter column referral_set_running_notional_taker_volume type text using (referral_set_running_notional_taker_volume::text);

alter table referral_set_stats
    alter column referral_set_running_notional_taker_volume set not null;


-- +goose Down

create view referral_set_referee_stats as
(
select set_id,
       at_epoch,
       referral_set_running_notional_taker_volume,
       stats.referee_stats ->> 'party_id'        as party_id,
       stats.referee_stats ->> 'discount_factor' as discount_factor,
       stats.referee_stats ->> 'reward_factor'   as reward_factor,
       vega_time
from referral_set_stats,
     jsonb_array_elements(referees_stats) with ordinality stats(referee_stats, position)
    );

