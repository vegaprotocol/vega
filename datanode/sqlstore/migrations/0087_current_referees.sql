-- +goose Up

CREATE UNIQUE INDEX referral_set_referees_pkey_update ON referral_set_referees(referral_set_id, referee, at_epoch);

ALTER TABLE referral_set_referees DROP CONSTRAINT referral_set_referees_pkey;

ALTER TABLE referral_set_referees
  ADD CONSTRAINT referral_set_referees_pkey PRIMARY KEY USING INDEX referral_set_referees_pkey_update;

CREATE VIEW current_referral_set_referees AS
SELECT DISTINCT ON (referee) *
FROM referral_set_referees
ORDER BY
  referee,
  at_epoch DESC;

-- +goose Down

DROP VIEW IF EXISTS current_referral_set_referees;

CREATE UNIQUE INDEX referral_set_referees_pkey_update ON referral_set_referees(referral_set_id, referee);

ALTER TABLE referral_set_referees DROP CONSTRAINT referral_set_referees_pkey;

ALTER TABLE referral_set_referees
  ADD CONSTRAINT referral_set_referees_pkey PRIMARY KEY USING INDEX referral_set_referees_pkey_update;

