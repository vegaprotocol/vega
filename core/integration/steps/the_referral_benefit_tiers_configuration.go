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

func TheReferralBenefitTiersConfiguration(config *referralcfg.Config, name string, table *godog.Table) error {
	benefitTiers := parseBenefitTiersConfigTable(table)

	config.BenefitTiersConfigs.Add(name, benefitTiers)
	return nil
}

func parseBenefitTiersConfigTable(table *godog.Table) []*types.BenefitTier {
	rows := StrictParseTable(table, []string{
		"minimum running notional taker volume",
		"minimum epochs",
		"referral reward factor",
		"referral discount factor",
	}, []string{})

	benefitTiers := make([]*types.BenefitTier, 0, len(rows))
	for _, row := range rows {
		specificRow := benefitTiersConfigRow{row: row}
		benefitTiers = append(benefitTiers, &types.BenefitTier{
			MinimumRunningNotionalTakerVolume: specificRow.MinimumRunningNotionalTakerVolume(),
			MinimumEpochs:                     specificRow.MinimumEpochs(),
			ReferralRewardFactor:              specificRow.ReferralRewardFactor(),
			ReferralDiscountFactor:            specificRow.ReferralDiscountFactor(),
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

func (r benefitTiersConfigRow) ReferralRewardFactor() num.Decimal {
	return r.row.MustDecimal("referral reward factor")
}

func (r benefitTiersConfigRow) ReferralDiscountFactor() num.Decimal {
	return r.row.MustDecimal("referral discount factor")
}
