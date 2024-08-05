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

func TheReferralBenefitTiersConfiguration(config *referralcfg.Config, name string, table *godog.Table) error {
	benefitTiers := parseBenefitTiersConfigTable(table)

	config.BenefitTiersConfigs.Add(name, benefitTiers)
	return nil
}

func parseBenefitTiersConfigTable(table *godog.Table) []*types.BenefitTier {
	rows := StrictParseTable(table, []string{
		"minimum running notional taker volume",
		"minimum epochs",
		"referral reward infra factor",
		"referral reward maker factor",
		"referral reward liquidity factor",
		"referral discount infra factor",
		"referral discount maker factor",
		"referral discount liquidity factor",
	}, []string{})

	benefitTiers := make([]*types.BenefitTier, 0, len(rows))
	for _, row := range rows {
		specificRow := benefitTiersConfigRow{row: row}
		benefitTiers = append(benefitTiers, &types.BenefitTier{
			MinimumRunningNotionalTakerVolume: specificRow.MinimumRunningNotionalTakerVolume(),
			MinimumEpochs:                     specificRow.MinimumEpochs(),
			ReferralRewardFactors: types.Factors{
				Infra:     specificRow.ReferralRewardInfraFactor(),
				Maker:     specificRow.ReferralRewardMakerFactor(),
				Liquidity: specificRow.ReferralRewardLiqFactor(),
			},
			ReferralDiscountFactors: types.Factors{
				Infra:     specificRow.ReferralDiscountInfraFactor(),
				Maker:     specificRow.ReferralDiscountMakerFactor(),
				Liquidity: specificRow.ReferralDiscountLiqFactor(),
			},
		})
	}

	return benefitTiers
}

type benefitTiersConfigRow struct {
	row RowWrapper
}

func (r benefitTiersConfigRow) MinimumRunningNotionalTakerVolume() *num.Uint {
	return r.row.MustUint("minimum running notional taker volume")
}

func (r benefitTiersConfigRow) MinimumEpochs() *num.Uint {
	return r.row.MustUint("minimum epochs")
}

func (r benefitTiersConfigRow) ReferralRewardInfraFactor() num.Decimal {
	return r.row.MustDecimal("referral reward infra factor")
}

func (r benefitTiersConfigRow) ReferralRewardMakerFactor() num.Decimal {
	return r.row.MustDecimal("referral reward maker factor")
}

func (r benefitTiersConfigRow) ReferralRewardLiqFactor() num.Decimal {
	return r.row.MustDecimal("referral reward liquidity factor")
}

func (r benefitTiersConfigRow) ReferralDiscountInfraFactor() num.Decimal {
	return r.row.MustDecimal("referral discount infra factor")
}

func (r benefitTiersConfigRow) ReferralDiscountMakerFactor() num.Decimal {
	return r.row.MustDecimal("referral discount maker factor")
}

func (r benefitTiersConfigRow) ReferralDiscountLiqFactor() num.Decimal {
	return r.row.MustDecimal("referral discount liquidity factor")
}
