-- +goose Up
CREATE TABLE IF NOT EXISTS party_margin_modes
(
  market_id                     BYTEA             NOT NULL,
  party_id                      BYTEA             NOT NULL,
  at_epoch                      BIGINT            NOT NULL,
  margin_mode                   INTEGER           NOT NULL,
  margin_factor                 NUMERIC(1000, 16) NULL,
  min_theoretical_margin_factor NUMERIC(1000, 16) NULL,
  max_theoretical_leverage      NUMERIC(1000, 16) NULL,
  PRIMARY KEY (market_id, party_id)
);

-- +goose Down

DROP TABLE IF EXISTS party_margin_modes;
