-- +goose Up

-- Carry over all reward totals that are missing across epochs.
-- +goose StatementBegin
do $$
declare
    first_epoch bigint;
    last_epoch bigint;
begin
    select min(epoch_id), max(epoch_id) into first_epoch, last_epoch from game_reward_totals;
    if first_epoch is null or last_epoch is null then
        return;
    end if;

    -- for each epoch, we want to carry over rewards that exist in this epoch, but not in the next
    -- we have to do it one epoch at a time, so that the carried forward rewards can continue to be carried
    -- over to the next epoch as well.
    for epoch in first_epoch..last_epoch-1 loop

        INSERT INTO game_reward_totals(game_id, party_id, asset_id, market_id, team_id, total_rewards, total_rewards_quantum, epoch_id)
        SELECT
            grto.game_id,
            grto.party_id,
            grto.asset_id,
            grto.market_id,
            grto.team_id,
            grto.total_rewards,
            grto.total_rewards_quantum,
            (grto.epoch_id + 1) AS epoch_id
        FROM game_reward_totals AS grto
            -- get the game end date from the transfer table and do not carry over rewards for games that have ended
        JOIN transfers t on grto.game_id = t.game_id and t.end_epoch > grto.epoch_id
        WHERE grto.epoch_id = epoch
        AND NOT EXISTS (
            SELECT 1
            FROM game_reward_totals AS grtc
            WHERE grtc.party_id = grto.party_id
            AND grtc.asset_id = grto.asset_id
            AND grtc.market_id = grto.market_id
            AND grtc.game_id = grto.game_id
            AND grtc.team_id = grto.team_id
            AND grtc.epoch_id = grto.epoch_id + 1
        );
    end loop;

end;
$$;
-- +goose StatementEnd



-- Create a function to carry over data between 2 given epochs.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION carry_over_rewards_on_epoch()
   RETURNS TRIGGER
   LANGUAGE PLPGSQL AS
$$
BEGIN
    INSERT INTO game_reward_totals (game_id, party_id, asset_id, market_id, team_id, total_rewards, total_rewards_quantum, epoch_id)
        SELECT
            grto.game_id,
            grto.party_id,
            grto.asset_id,
            grto.market_id,
            grto.team_id,
            grto.total_rewards,
            grto.total_rewards_quantum,
            (NEW.id - 1) AS epoch_id
        FROM game_reward_totals AS grto
         -- get the game end date from the transfer table and do not carry over rewards for games that have ended
         JOIN transfers t on grto.game_id = t.game_id and t.end_epoch > grto.epoch_id
        WHERE grto.epoch_id = (NEW.id - 2)
        AND NOT EXISTS (
            SELECT 1
            FROM game_reward_totals AS grtc
            WHERE grtc.party_id =  grto.party_id
            AND grtc.game_id = grto.game_id
            AND grtc.asset_id = grto.asset_id
            AND grtc.market_id = grto.market_id
            AND grtc.team_id = grto.team_id
            AND grtc.epoch_id = (NEW.id - 1)
        );
RETURN NEW;
END;
$$;
-- +goose StatementEnd

-- add trigger to the epochs table
CREATE OR REPLACE TRIGGER carry_over_epoch_data
    AFTER INSERT
    ON epochs
    FOR EACH STATEMENT
    EXECUTE FUNCTION carry_over_rewards_on_epoch();

-- +goose Down

DROP TRIGGER IF EXISTS carry_over_epoch_data ON epochs;

DROP FUNCTION IF EXISTS carry_over_rewards_on_epoch();
