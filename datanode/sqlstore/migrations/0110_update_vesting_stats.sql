-- +goose Up

DROP TRIGGER IF EXISTS update_party_vesting_stats ON party_vesting_stats;
-- update the vesting stats table to include the summed quantum balance and reward bonus multiplier
ALTER TABLE party_vesting_stats
ADD COLUMN IF NOT EXISTS summed_reward_bonus_multiplier NUMERIC(1000, 16) NOT NULL DEFAULT 0,
ADD COLUMN IF NOT EXISTS summed_quantum_balance NUMERIC(1000, 16) NOT NULL DEFAULT 0;

ALTER TABLE party_vesting_stats_current
ADD COLUMN IF NOT EXISTS summed_reward_bonus_multiplier NUMERIC(1000, 16) NOT NULL DEFAULT 0,
ADD COLUMN IF NOT EXISTS summed_quantum_balance NUMERIC(1000, 16) NOT NULL DEFAULT 0;

-- create the trigger functions and triggers
-- +goose StatementBegin
create or replace function update_party_vesting_stats()
       returns trigger
       language plpgsql
as $$
   begin
        insert into party_vesting_stats_current(party_id, at_epoch, reward_bonus_multiplier, quantum_balance, summed_reward_bonus_multiplier, summed_quantum_balance, vega_time)
        values (new.party_id, new.at_epoch, new.reward_bonus_multiplier, new.quantum_balance, new.summed_reward_bonus_multiplier, new.summed_quantum_balance, new.vega_time)
        on conflict(party_id)
        do update set
            at_epoch = excluded.at_epoch,
            reward_bonus_multiplier = excluded.reward_bonus_multiplier,
            quantum_balance = excluded.quantum_balance,
            summed_reward_bonus_multiplier = excluded.summed_reward_bonus_multiplier,
            summed_quantum_balance = excluded.summed_quantum_balance,
            vega_time = excluded.vega_time;
        return null;
   end;
$$;
-- +goose StatementEnd

CREATE TRIGGER update_party_vesting_stats
    after insert or update
    on party_vesting_stats
    for each row execute function update_party_vesting_stats();

-- +goose Down

DROP TRIGGER IF EXISTS update_party_vesting_stats ON party_vesting_stats;
ALTER TABLE party_vesting_stats
DROP COLUMN summed_reward_bonus_multiplier,
DROP COLUMN summed_quantum_balance;

ALTER TABLE party_vesting_stats_current
DROP COLUMN summed_reward_bonus_multiplier,
DROP COLUMN summed_quantum_balance;

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

CREATE TRIGGER update_party_vesting_stats
    after insert or update
    on party_vesting_stats
    for each row execute function update_party_vesting_stats();
