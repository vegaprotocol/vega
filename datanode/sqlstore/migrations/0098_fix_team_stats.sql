-- +goose Up

DROP TRIGGER IF EXISTS update_teams_stats ON rewards;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_teams_stats()
    RETURNS
        TRIGGER
    LANGUAGE plpgsql
AS
$$
DECLARE
    party_team_id          BYTEA;
    additional_game_id     JSONB;
    team_dispatch_strategy JSONB;
BEGIN
    -- Exclude any reward that is not associated to a game, as we only account for
    -- game rewards in teams.
    IF new.game_id IS NULL THEN
        RETURN NULL;
    END IF;

    -- Get the dispatch strategy for the game
    SELECT dispatch_strategy
    INTO team_dispatch_strategy
    FROM (SELECT DISTINCT
        ON (game_id) game_id,
                     dispatch_strategy
          FROM transfers
          WHERE transfer_type = 'Recurring'
          ORDER BY game_id, vega_time DESC) AS dispatch_strategies
    WHERE game_id = NEW.game_id;

    -- Exclude non team rewards.
    IF team_dispatch_strategy IS NULL THEN
        RETURN NULL;
    ELSIF (team_dispatch_strategy ->> 'entity_scope' != '2') THEN
        RETURN NULL;
    END IF;

    WITH current_team_members AS (SELECT DISTINCT ON (party_id) *
                                  FROM team_members
                                  ORDER BY party_id,
                                           joined_at_epoch DESC)
    SELECT team_id
    INTO party_team_id
    FROM current_team_members
    WHERE party_id = new.party_id;

    -- If the party does not belong to a team, no reporting needs to be done.
    IF party_team_id IS NULL THEN
        RETURN NULL;
    END IF;

    additional_game_id = JSONB_BUILD_OBJECT(new.game_id, TRUE);

    INSERT INTO teams_stats (team_id, party_id, at_epoch, total_quantum_volume, total_quantum_reward, games_played)
    VALUES (party_team_id, new.party_id, new.epoch_id, 0, new.quantum_amount, additional_game_id)
    ON CONFLICT (team_id, party_id, at_epoch) DO UPDATE
        SET total_quantum_reward = teams_stats.total_quantum_reward + new.quantum_amount,
            games_played         = teams_stats.games_played || additional_game_id;
    RETURN NULL;
END;
$$;
-- +goose StatementEnd

CREATE TRIGGER update_teams_stats
    AFTER INSERT
    ON rewards
    FOR EACH ROW
EXECUTE FUNCTION update_teams_stats();

-- +goose Down

DROP TRIGGER IF EXISTS update_teams_stats ON rewards;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_teams_stats()
    RETURNS
        TRIGGER
    LANGUAGE plpgsql
AS
$$
DECLARE
    party_team_id          BYTEA;
    additional_game_id     JSONB;
    team_dispatch_strategy JSONB;
BEGIN
    -- Exclude any reward that is not associated to a game, as we only account for
    -- game rewards in teams.
    IF new.game_id IS NULL THEN
        RETURN NULL;
    END IF;

    -- Get the dispatch strategy for the game
    SELECT dispatch_strategy
    INTO team_dispatch_strategy
    FROM (SELECT DISTINCT
        ON (game_id) game_id,
                     dispatch_strategy
          FROM transfers
          WHERE transfer_type = 'Recurring'
          ORDER BY game_id, vega_time DESC) AS dispatch_strategies
    WHERE game_id = NEW.game_id;

    IF team_dispatch_strategy IS NULL THEN
        RETURN NULL;
    ELSIF (team_dispatch_strategy ->> 'entity_scope' != '2') THEN
        RETURN NULL;
    END IF;

    WITH current_team_members AS (SELECT DISTINCT
        ON (party_id) *
                                  FROM team_members
                                  ORDER BY party_id,
                                           joined_at_epoch DESC)
    SELECT team_id
    INTO party_team_id
    FROM current_team_members
    WHERE party_id = new.party_id;

    -- If the party does not belong to a team, no reporting needs to be done.
    IF party_team_id IS NULL THEN
        RETURN NULL;
    END IF;

    additional_game_id = JSONB_BUILD_OBJECT(new.game_id, TRUE);

    INSERT INTO teams_stats (team_id, party_id, at_epoch, total_quantum_volume, total_quantum_reward, games_played)
    VALUES (party_team_id, new.party_id, new.epoch_id, 0, new.quantum_amount, additional_game_id)
    ON CONFLICT (team_id, party_id, at_epoch) DO UPDATE
        SET total_quantum_reward = teams_stats.total_quantum_reward + new.quantum_amount,
            games_played         = teams_stats.games_played || additional_game_id;
    RETURN NULL;
END;
$$;
-- +goose StatementEnd
