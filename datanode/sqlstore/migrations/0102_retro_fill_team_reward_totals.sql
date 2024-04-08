-- +goose Up

-- Carry over all reward totals that are missing across epochs.
INSERT INTO game_reward_totals (game_id, party_id, asset_id, market_id, team_id, total_rewards, total_rewards_quantum, epoch_id)
    SELECT
        grto.game_id,
        grto.party_id,
        grto.asset_id,
        grto.market_id,
        grto.team_id,
        grto.total_rewards,
        grto.total_rewards_quantum,
        (epoch_id + 1) AS epoch_id
    FROM game_reward_totals AS grto
    WHERE grto.epoch_id < (SELECT MAX(id)-1 FROM epochs)
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

-- Create a function to carry over data between 2 given epochs.
CREATE OR REPLACE FUNCTION carry_over_rewards_on_epoch(IN previous_epoch BIGINT, IN ending_epoch BIGINT)
    LANGUAGE plpgsql AS
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
            ending_epoch AS epoch_id
        FROM game_reward_totals AS grto
        WHERE grto.epoch_id = previous_epoch
        AND NOT EXISTS (
            SELECT 1
            FROM game_reward_totals AS grtc
            WHERE grtc.party_id =  grto.party_id
            AND grtc.game_id = grto.game_id
            AND grtc.asset_id = grto.asset_id
            AND grtc.market_id = grto.market_id
            AND grtc.team_id = grto.team_id
            AND grtc.epoch_id = ending_epoch
        );
END;
$$;

-- add trigger to the epochs table
CREATE OR REPLACE TRIGGER carry_over_epoch_data
    AFTER INSERT
    ON epochs
    FOR EACH STATEMENT
    EXECUTE FUNCTION carry_over_rewards_on_epoch(NEW.id - 2, NEW.id - 1);

-- +goose Down

DROP TRIGGER IF EXISTS carry_over_epoch_data ON epochs;

DROP FUNCTION IF EXISTS carry_over_rewards_on_epoch(BIGINT, BIGINT);
