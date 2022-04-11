package steps

import (
	"strconv"

	"code.vegaprotocol.io/vega/integration/steps/market"
)

func OracleSpecSettlementPriceDecimals(config *market.Config, name string, settlementDP string) error {
	dp, err := strconv.ParseUint(settlementDP, 10, 0)
	if err != nil {
		return err
	}
	config.OracleConfigs.SetSettlementPriceDP(name, uint32(dp))
	return nil
}
