-- +goose Up

-- paid_liquidity_fees stores the per-epoch accumulated paid fees
create table if not exists paid_liquidity_fees (
      market_id bytea not null,
      asset_id bytea not null,
      epoch_seq bigint not null,
      total_fees_paid text not null,
      fees_paid_per_party jsonb not null,
      primary key (market_id, asset_id, epoch_seq)
);

-- +goose Down
drop table if exists paid_liquidity_fees;
