package steps

import (
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/logging"
)

func DebugMarketData(
	exec *execution.Engine,
	log *logging.Logger,
	market string,
) error {
	log.Info("DUMPING ORDERS")
	marketData, err := exec.GetMarketData(market)
	if err != nil {
		return errors.New("market not found")
	}
	log.Info(fmt.Sprintf("%s", marketData.String()))

	return nil
}
