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

package steps

import (
	"fmt"
	"time"

	referralcfg "code.vegaprotocol.io/vega/core/integration/steps/referral"
	"code.vegaprotocol.io/vega/core/referral"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"github.com/cucumber/godog"

	"code.vegaprotocol.io/vega/core/types"
)

func TheReferralProgram(referralProgramConfig *referralcfg.Config, referralProgramEngine *referral.Engine, table *godog.Table) error {
	row := parseReferralProgram(table)

	benefitTiersConfigName := row.BenefitTiers()
	benefitTiers, err := referralProgramConfig.BenefitTiersConfigs.Get(benefitTiersConfigName)
	if err != nil {
		return fmt.Errorf("could not load benefit tiers configuration %q: %w", benefitTiersConfigName, err)
	}

	stakingTiersConfigName := row.StakingTiers()
	stakingTiers, err := referralProgramConfig.StakingTiersConfigs.Get(stakingTiersConfigName)
	if err != nil {
		return fmt.Errorf("could not load staking tiers configuration %q: %w", stakingTiersConfigName, err)
	}

	referralProgramEngine.UpdateProgram(&types.ReferralProgram{
		ID:                    vgcrypto.RandomHash(),
		EndOfProgramTimestamp: row.EndOfProgram(),
		WindowLength:          row.WindowLength(),
		BenefitTiers:          benefitTiers,
		StakingTiers:          stakingTiers,
	})

	return nil
}

func parseReferralProgram(table *godog.Table) parseReferralProgramTable {
	row := StrictParseFirstRow(table, []string{
		"end of program",
		"window length",
		"benefit tiers",
		"staking tiers",
	}, []string{
		"decimal places",
		"position decimal places",
	})
	return parseReferralProgramTable{
		row: row,
	}
}

type parseReferralProgramTable struct {
	row RowWrapper
}

func (r parseReferralProgramTable) EndOfProgram() time.Time {
	return r.row.MustTime("end of program")
}

func (r parseReferralProgramTable) WindowLength() uint64 {
	return r.row.MustU64("window length")
}

func (r parseReferralProgramTable) BenefitTiers() string {
	return r.row.MustStr("benefit tiers")
}

func (r parseReferralProgramTable) StakingTiers() string {
	return r.row.MustStr("staking tiers")
}
