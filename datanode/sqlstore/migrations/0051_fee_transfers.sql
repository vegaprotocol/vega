-- +goose Up

-- paid_liquidity_fees stores the per-epoch accumulated paid fees
create table if not exists transfer_fees (
      transfer_id bytea not null,
      asset_id bytea not null,
      party_id bytea not null,
      amount HUGEINT not null,
      vega_time timestamp with time zone not null,
      primary key (vega_time, party_id, asset_id)
);

create index transfer_fees_transfer_id_idx on transfer_fees(transfer_id);
create index transfer_fees_party_id_idx on transfer_fees(party_id);

-- +goose Down
drop index transfer_fees_transfer_id_idx;
drop index transfer_fees_party_id_idx;
drop table if exists transfer_fees;
