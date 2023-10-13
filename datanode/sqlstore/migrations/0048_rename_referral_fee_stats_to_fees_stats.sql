-- +goose Up

alter table referral_fee_stats
    rename to fees_stats;
alter index referral_fee_stats_pkey rename to fees_stats_pkey;
alter index referral_fee_stats_vega_time_idx rename to fees_stats_vega_time_idx;

-- +goose Down

alter table fees_stats
    rename to referral_fee_stats;
alter index fees_stats_pkey rename to referral_fee_stats_pkey;
alter index fees_stats_vega_time_idx rename to referral_fee_stats_vega_time_idx;
