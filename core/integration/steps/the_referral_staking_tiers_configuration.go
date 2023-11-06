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
