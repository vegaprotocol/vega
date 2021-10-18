package products

import (
	"context"
	"strconv"

	"code.vegaprotocol.io/vega/oracles"
)

// SetSettlementPrice is a backdoor helper function that updates settlement price as if it had come from an oracle.
func (f *Future) SetSettlementPrice(ctx context.Context, priceName string, settlementPrice uint64) error {
	od := oracles.OracleData{Data: map[string]string{}}
	od.Data[priceName] = strconv.FormatUint(settlementPrice, 10)
	return f.updateSettlementPrice(ctx, od)
}

// SetTradingTerminated is a backdoor helper function that updates tradingTerminated as if it had come from an oracle.
func (f *Future) SetTradingTerminated(ctx context.Context, tradingTerminated bool) error {
	od := oracles.OracleData{Data: map[string]string{}}
	od.Data["trading.terminated"] = strconv.FormatBool(tradingTerminated)
	return f.updateTradingTerminated(ctx, od)
}
