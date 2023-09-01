package sqlstore

import (
	"context"
	"time"

	"github.com/georgysavva/scany/pgxscan"

	"code.vegaprotocol.io/vega/datanode/metrics"

	"code.vegaprotocol.io/vega/datanode/entities"
)

type ReferralPrograms struct {
	*ConnectionSource
}

func NewReferralPrograms(connectionSource *ConnectionSource) *ReferralPrograms {
	return &ReferralPrograms{
		ConnectionSource: connectionSource,
	}
}

func (rp *ReferralPrograms) AddReferralProgram(ctx context.Context, referral *entities.ReferralProgram) error {
	defer metrics.StartSQLQuery("ReferralPrograms", "AddReferralProgram")()
	return rp.insertReferralProgram(ctx, referral)
}

func (rp *ReferralPrograms) insertReferralProgram(ctx context.Context, referral *entities.ReferralProgram) error {
	_, err := rp.Connection.Exec(ctx,
		`INSERT INTO referral_programs (id, version, benefit_tiers, end_of_program_timestamp, window_length, staking_tiers, vega_time, ended_at)
    		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		referral.ID,
		referral.Version,
		referral.BenefitTiers,
		referral.EndOfProgramTimestamp,
		referral.WindowLength,
		referral.StakingTiers,
		referral.VegaTime,
		referral.EndedAt,
	)
	return err
}

func (rp *ReferralPrograms) UpdateReferralProgram(ctx context.Context, referral *entities.ReferralProgram) error {
	defer metrics.StartSQLQuery("ReferralPrograms", "UpdateReferralProgram")()
	return rp.insertReferralProgram(ctx, referral)
}

func (rp *ReferralPrograms) EndReferralProgram(ctx context.Context, referralID entities.ReferralID, version uint64,
	vegaTime time.Time,
) error {
	defer metrics.StartSQLQuery("ReferralPrograms", "EndReferralProgram")()
	_, err := rp.Connection.Exec(ctx,
		`INSERT INTO referral_programs (id, version, benefit_tiers, end_of_program_timestamp, window_length, staking_tiers, vega_time, ended_at)
            SELECT id, $1, benefit_tiers, end_of_program_timestamp, window_length, staking_tiers, $2, $2
            FROM current_referral_program`, version, vegaTime,
	)

	return err
}

func (rp *ReferralPrograms) GetCurrentReferralProgram(ctx context.Context) (entities.ReferralProgram, error) {
	defer metrics.StartSQLQuery("ReferralPrograms", "GetCurrentReferralProgram")()
	var referralProgram entities.ReferralProgram

	query := `SELECT id, version, benefit_tiers, end_of_program_timestamp, window_length, staking_tiers, vega_time, ended_at FROM current_referral_program`
	if err := pgxscan.Get(ctx, rp.Connection, &referralProgram, query); err != nil {
		return referralProgram, err
	}

	return referralProgram, nil
}
