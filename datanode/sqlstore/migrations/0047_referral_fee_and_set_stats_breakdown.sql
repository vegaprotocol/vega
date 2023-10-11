-- +goose Up

-- index the json columns to make them faster to query
create index if not exists idx_referral_fee_stats_total_rewards_paid on referral_fee_stats using gin(total_rewards_paid);
create index if not exists idx_referral_fee_stats_referrer_rewards_generated on referral_fee_stats using gin(referrer_rewards_generated);
create index if not exists idx_referral_fee_stats_referees_discount_applied on referral_fee_stats using gin(referees_discount_applied);
create index if not exists idx_referral_fee_stats_volume_discount_applied on referral_fee_stats using gin(volume_discount_applied);

-- +goose Down

drop index if exists idx_referral_fee_stats_total_rewards_paid;
drop index if exists idx_referral_fee_stats_referrer_rewards_generated;
drop index if exists idx_referral_fee_stats_referees_discount_applied;
drop index if exists idx_referral_fee_stats_volume_discount_applied;
