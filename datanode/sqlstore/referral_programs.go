// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package sqlstore

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	"code.vegaprotocol.io/vega/libs/ptr"

	"github.com/georgysavva/scany/pgxscan"
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
	if len(referral.BenefitTiers) > 0 && referral.BenefitTiers[0].TierNumber == nil {
		// update stores to set tier numbers.
		for i := range referral.BenefitTiers {
			referral.BenefitTiers[i].TierNumber = ptr.From(uint64(i + 1))
		}
	}
	_, err := rp.Exec(ctx,
		`INSERT INTO referral_programs (id, version, benefit_tiers, end_of_program_timestamp, window_length, staking_tiers, vega_time, seq_num)
    		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		referral.ID,
		referral.Version,
		referral.BenefitTiers,
		referral.EndOfProgramTimestamp,
		referral.WindowLength,
		referral.StakingTiers,
		referral.VegaTime,
		referral.SeqNum,
	)
	return err
}

func (rp *ReferralPrograms) UpdateReferralProgram(ctx context.Context, referral *entities.ReferralProgram) error {
	defer metrics.StartSQLQuery("ReferralPrograms", "UpdateReferralProgram")()
	return rp.insertReferralProgram(ctx, referral)
}

func (rp *ReferralPrograms) EndReferralProgram(ctx context.Context, version uint64, endedAt time.Time, vegaTime time.Time, seqNum uint64) error {
	defer metrics.StartSQLQuery("ReferralPrograms", "EndReferralProgram")()
	_, err := rp.Exec(ctx,
		`INSERT INTO referral_programs (id, version, benefit_tiers, end_of_program_timestamp, window_length, staking_tiers, ended_at, vega_time, seq_num)
            SELECT id, $1, benefit_tiers, end_of_program_timestamp, window_length, staking_tiers, $2, $3, $4
            FROM current_referral_program`, version, endedAt, vegaTime, seqNum,
	)

	return err
}

func (rp *ReferralPrograms) GetCurrentReferralProgram(ctx context.Context) (entities.ReferralProgram, error) {
	var referralProgram entities.ReferralProgram
	defer func() {
		// ensure the tier numbers are set
		if len(referralProgram.BenefitTiers) > 0 && referralProgram.BenefitTiers[0].TierNumber == nil {
			for i := range referralProgram.BenefitTiers {
				referralProgram.BenefitTiers[i].TierNumber = ptr.From(uint64(i + 1))
			}
		}
		metrics.StartSQLQuery("ReferralPrograms", "GetCurrentReferralProgram")()
	}()

	query := `SELECT id, version, benefit_tiers, end_of_program_timestamp, window_length, staking_tiers, vega_time, ended_at, seq_num FROM current_referral_program`
	if err := pgxscan.Get(ctx, rp.ConnectionSource, &referralProgram, query); err != nil {
		return referralProgram, err
	}

	return referralProgram, nil
}
