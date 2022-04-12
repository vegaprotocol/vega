package steps

import (
	"strconv"

	"code.vegaprotocol.io/vega/integration/steps/market"
)

func OracleSpecSettlementPriceDecimalScalingFactorExponent(config *market.Config, name string, exponent string) error {
	dp, err := strconv.ParseUint(exponent, 10, 0)
	if err != nil {
		return err
	}
	config.OracleConfigs.SetSettlementPriceDecimalScalingExponent(name, int32(dp))
	return nil
}
