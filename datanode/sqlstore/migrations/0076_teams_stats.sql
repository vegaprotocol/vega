-- +goose Up
CREATE TABLE IF NOT EXISTS teams_stats
(
  team_id              BYTEA             NOT NULL,
  party_id             BYTEA             NOT NULL,
  at_epoch             BIGINT            NOT NULL,
  total_volume         NUMERIC(1000, 16) NOT NULL,
  total_quantum_reward NUMERIC(1000, 16) NOT NULL,
  games_played         JSONB             NOT NULL,
  PRIMARY KEY (team_id, party_id, at_epoch)
);

CREATE INDEX idx_teams_stats_at_epoch ON teams_stats (at_epoch DESC);

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION update_teams_stats()
  RETURNS TRIGGER
  LANGUAGE plpgsql AS
$$
DECLARE
  party_team_id      BYTEA;
  additional_game_id JSONB;

BEGIN

  WITH
    current_team_members AS (
      SELECT DISTINCT ON (party_id) *
      FROM team_members
      ORDER BY
        party_id,
        joined_at_epoch DESC
    )
  SELECT team_id
  INTO party_team_id
  FROM current_team_members
  WHERE party_id = new.party_id;

  -- If the party does not belong to a team, no reporting needs to be done.
  IF party_team_id IS NULL THEN
    RETURN NULL;
  END IF;

  -- This build a JSON object so we can merge its content into any existing record,
  -- using the concatenation operator "||".
  -- Building an object, and not an array, is important for the concatenation
  -- operator to deduplicate the keys. We don't want to insert the same game ID
  -- several times.
  IF new.game_id IS NOT NULL THEN
    -- Create an object as: { "GAME_ID": true }.
    -- The value "true" is just a filler.
    additional_game_id = JSONB_BUILD_OBJECT(new.game_id, TRUE);
  ELSE
    -- No game ID, then create an empty object.
    additional_game_id = JSONB_BUILD_OBJECT();
  END IF;

  INSERT INTO teams_stats(team_id, party_id, at_epoch, total_volume, total_quantum_reward, games_played)
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
DROP FUNCTION IF EXISTS update_teams_stats;

DROP TABLE IF EXISTS teams_stats;
