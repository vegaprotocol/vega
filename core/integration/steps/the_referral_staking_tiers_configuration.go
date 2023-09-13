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
	referralcfg "code.vegaprotocol.io/vega/core/integration/steps/referral"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"github.com/cucumber/godog"
)

func TheReferralStakingTiersConfiguration(config *referralcfg.Config, name string, table *godog.Table) error {
	stakingTiers := parseStakingTiersConfigTable(table)

	config.StakingTiersConfigs.Add(name, stakingTiers)
	return nil
}

func parseStakingTiersConfigTable(table *godog.Table) []*types.StakingTier {
	rows := StrictParseTable(table, []string{
		"minimum staked tokens",
		"referral reward multiplier",
	}, []string{})

	stakingTiers := make([]*types.StakingTier, 0, len(rows))
	for _, row := range rows {
		specificRow := stakingTiersConfigRow{row: row}
		stakingTiers = append(stakingTiers, &types.StakingTier{
			MinimumStakedTokens:      specificRow.MinimumStakedTokens(),
			ReferralRewardMultiplier: specificRow.ReferralRewardMultiplier(),
		})
	}

	return stakingTiers
}

type stakingTiersConfigRow struct {
	row RowWrapper
}

func (r stakingTiersConfigRow) MinimumStakedTokens() *num.Uint {
	return r.row.MustUint("minimum staked tokens")
}

func (r stakingTiersConfigRow) ReferralRewardMultiplier() num.Decimal {
	return r.row.MustDecimal("referral reward multiplier")
}
