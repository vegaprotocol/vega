-- +goose Up

-- paid_liquidity_fees stores the per-epoch accumulated paid fees
create table if not exists paid_liquidity_fees (
      market_id bytea not null,
      asset_id bytea not null,
      epoch_seq bigint not null,
      total_fees_paid text not null,
      fees_paid_per_party jsonb not null,
      vega_time timestamp with time zone not null,
      primary key (vega_time, market_id, asset_id, epoch_seq)
);

create index paid_liquidity_fees_market_id_idx on paid_liquidity_fees(market_id);
create index paid_liquidity_fees_asset_id_idx on paid_liquidity_fees(asset_id);
create index paid_liquidity_fees_epoch_seq_idx on paid_liquidity_fees(epoch_seq);
create index paid_liquidity_fees_fees_paid_per_party_ix on paid_liquidity_fees((fees_paid_per_party->>'party'));

-- +goose Down
drop index paid_liquidity_fees_market_id_idx;
drop index paid_liquidity_fees_asset_id_idx;
drop index paid_liquidity_fees_epoch_seq_idx;
drop index paid_liquidity_fees_fees_paid_per_party_ix;
drop table if exists paid_liquidity_fees;