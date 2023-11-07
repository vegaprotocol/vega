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
	LiquiditySLAParams  *slaParams
	LiquidationStrat    *liquidationConfig
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
	LiquidationStrat    *liquidationConfig
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
		LiquiditySLAParams:  newLiquiditySLAParams(unmarshaler),
		LiquidationStrat:    newLiquidationConfig(unmarshaler),
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
		LiquidationStrat:    newLiquidationConfig(u),
	}
}
