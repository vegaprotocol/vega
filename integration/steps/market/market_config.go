package market

import "code.vegaprotocol.io/vega/integration/steps/market/defaults"

type Config struct {
	RiskModels        *riskModels
	FeesConfig        *feesConfig
	OracleConfigs     *oracleConfigs
	PriceMonitoring   *priceMonitoring
	MarginCalculators *marginCalculators
}

func NewMarketConfig() *Config {
	unmarshaler := defaults.NewUnmarshaler()
	return &Config{
		RiskModels:        newRiskModels(unmarshaler),
		FeesConfig:        newFeesConfig(unmarshaler),
		OracleConfigs:     newOracleSpecs(unmarshaler),
		PriceMonitoring:   newPriceMonitoring(unmarshaler),
		MarginCalculators: newMarginCalculators(unmarshaler),
	}
}
