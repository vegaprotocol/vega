-- +goose Up

-- +goose StatementBegin
do $$
begin
    if not exists (select * from timescaledb_information.hypertables where hypertable_name = 'fees_stats_by_party') then
        perform create_hypertable('fees_stats_by_party', 'vega_time', chunk_time_interval => INTERVAL '1 day', migrate_data => true);
    end if;
end $$;
-- +goose StatementEnd

-- +goose StatementBegin
do $$
begin
    if not exists (select * from timescaledb_information.hypertables where hypertable_name = 'paid_liquidity_fees') then
        perform create_hypertable('paid_liquidity_fees', 'vega_time', chunk_time_interval => INTERVAL '1 day', migrate_data => true);
    end if;
end $$;
-- +goose StatementEnd

-- +goose Down

-- nothing to do, we're not going to convert it back
