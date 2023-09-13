// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
