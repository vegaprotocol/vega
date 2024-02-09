-- +goose Up

ALTER TABLE proposals
      ADD COLUMN batch_id BYTEA,
      ADD COLUMN batch_terms JSONB DEFAULT '{}' NOT NULL,
      ALTER COLUMN terms DROP NOT NULL;

CREATE INDEX ON proposals (batch_id);
ALTER TYPE proposal_error ADD VALUE IF NOT EXISTS 'PROPOSAL_ERROR_PROPOSAL_IN_BATCH_REJECTED';
ALTER TYPE proposal_error ADD VALUE IF NOT EXISTS 'PROPOSAL_ERROR_PROPOSAL_IN_BATCH_DECLINED';

CREATE OR REPLACE VIEW proposals_current AS (
  SELECT DISTINCT ON (id) * FROM proposals ORDER BY id, vega_time DESC
);

ALTER TABLE votes ADD COLUMN per_market_equity_like_share_weight JSONB;

CREATE OR REPLACE VIEW votes_current AS (
  SELECT DISTINCT ON (proposal_id, party_id) * FROM votes ORDER BY proposal_id, party_id, vega_time DESC
);

-- +goose Down

DROP INDEX IF EXISTS proposals_idx_batch_id;
DROP VIEW proposals_current;
ALTER TABLE proposals
      DROP COLUMN IF EXISTS batch_id,
      DROP COLUMN IF EXISTS batch_terms;

CREATE OR REPLACE VIEW proposals_current AS (
  SELECT DISTINCT ON (id) * FROM proposals ORDER BY id, vega_time DESC
);

DROP VIEW votes_current;
ALTER TABLE votes DROP COLUMN per_market_equity_like_share_weight;

CREATE OR REPLACE VIEW votes_current AS (
  SELECT DISTINCT ON (proposal_id, party_id) * FROM votes ORDER BY proposal_id, party_id, vega_time DESC
);
