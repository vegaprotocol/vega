-- +goose Up

-- create the history and current tables for parties vestng stats
create table if not exists party_vesting_stats (
       party_id bytea not null,
       at_epoch bigint not null,
       reward_bonus_multiplier NUMERIC(1000, 16) not null,
       quantum_balance NUMERIC(1000, 16) not null,
       vega_time timestamp with time zone not null,
       primary key (vega_time, party_id)
);

select create_hypertable('party_vesting_stats', 'vega_time', chunk_time_interval => INTERVAL '1 day');

create table if not exists party_vesting_stats_current (
       party_id bytea not null,
       at_epoch bigint not null,
       reward_bonus_multiplier NUMERIC(1000, 16) not null,
       quantum_balance NUMERIC(1000, 16) not null,
       vega_time timestamp with time zone not null,
       primary key (party_id)
);

-- create the trigger functions and triggers
-- +goose StatementBegin
create or replace function update_party_vesting_stats()
       returns trigger
       language plpgsql
as $$
   begin
        insert into party_vesting_stats_current(party_id, at_epoch, reward_bonus_multiplier, quantum_balance, vega_time)
        values (new.party_id, new.at_epoch, new.reward_bonus_multiplier, new.quantum_balance, new.vega_time)
        on conflict(party_id)
        do update set
           at_epoch = excluded.at_epoch,
           reward_bonus_multiplier = excluded.reward_bonus_multiplier,
	   quantum_balance = excluded.quantum_balance,
	   vega_time = excluded.vega_time;
        return null;
   end;
$$;
-- +goose StatementEnd

create trigger update_party_vesting_stats
    after insert or update
    on party_vesting_stats
    for each row execute function update_party_vesting_stats();

-- +goose Down

drop table if exists party_vesting_stats_current;
drop table if exists party_vesting_stats;

drop function if exists update_party_vesting_stats;
