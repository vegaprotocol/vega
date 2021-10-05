package steps

import (
	"errors"

	"code.vegaprotocol.io/vega/logging"
)

func DebugMarketData(
	exec Execution,
	log *logging.Logger,
	market string,
) error {
	log.Info("DUMPING MARKET DATA")
	marketData, err := exec.GetMarketData(market)
	if err != nil {
		return errors.New("market not found")
	}
	log.Info(marketData.String())

	return nil
}
