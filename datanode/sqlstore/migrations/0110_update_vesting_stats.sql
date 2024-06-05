-- +goose Up

-- update the vesting stats table to include the summed quantum balance and reward bonus multiplier
ALTER TABLE party_vesting_stats
ADD COLUMN IF NOT EXISTS summed_reward_bonus_multiplier NUMERIC(1000, 16) NOT NULL,
ADD COLUMN IF NOT EXISTS summed_quantum_balance NUMERIC(1000, 16) NOT NULL;

ALTER TABLE party_vesting_stats_current
ADD COLUMN IF NOT EXISTS summed_reward_bonus_multiplier NUMERIC(1000, 16) NOT NULL,
ADD COLUMN IF NOT EXISTS summed_quantum_balance NUMERIC(1000, 16) NOT NULL;

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

create or replace trigger update_party_vesting_stats after
insert or update on party_vesting_stats for each row
execute function update_party_vesting_stats ();

-- +goose Down

ALTER TABLE party_vesting_stats
DROP COLUMN summed_reward_bonus_multiplier,
DROP COLUMN summed_quantum_balance;

ALTER TABLE party_vesting_stats_current
DROP COLUMN summed_reward_bonus_multiplier,
DROP COLUMN summed_quantum_balance;

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

create or replace trigger update_party_vesting_stats after
insert or update on party_vesting_stats for each row
execute function update_party_vesting_stats ();