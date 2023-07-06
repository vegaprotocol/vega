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

package market

import (
	"code.vegaprotocol.io/vega/core/integration/steps/market/defaults"
	"code.vegaprotocol.io/vega/libs/num"
)

type Config struct {
	RiskModels          *riskModels
	FeesConfig          *feesConfig
	OracleConfigs       *oracleConfigs
	PriceMonitoring     *priceMonitoring
	MarginCalculators   *marginCalculators
	LiquidityMonitoring *liquidityMonitoring
}

type SuccessorConfig struct {
	ParentID            string
	InsuranceFraction   num.Decimal
	PriceMonitoring     *priceMonitoring
	LiquidityMonitoring *liquidityMonitoring
	RiskModels          *riskModels
	PositionDecimals    int64
	Decimals            uint64
	PriceRange          num.Decimal
	LinSlip             num.Decimal
	QuadSlip            num.Decimal
}

func NewMarketConfig() *Config {
	unmarshaler := defaults.NewUnmarshaler()
	return &Config{
		RiskModels:          newRiskModels(unmarshaler),
		FeesConfig:          newFeesConfig(unmarshaler),
		OracleConfigs:       newOracleSpecs(unmarshaler),
		PriceMonitoring:     newPriceMonitoring(unmarshaler),
		MarginCalculators:   newMarginCalculators(unmarshaler),
		LiquidityMonitoring: newLiquidityMonitoring(unmarshaler),
	}
}

func NewSuccessorConfig() *SuccessorConfig {
	u := defaults.NewUnmarshaler()
	zero := num.DecimalZero()
	return &SuccessorConfig{
		InsuranceFraction:   zero,
		PriceMonitoring:     newPriceMonitoring(u),
		LiquidityMonitoring: newLiquidityMonitoring(u),
		RiskModels:          newRiskModels(u),
		PriceRange:          zero,
		LinSlip:             zero,
		QuadSlip:            zero,
	}
}
