package products

import (
	"context"
	"strconv"

	"code.vegaprotocol.io/vega/oracles"
)

func (f *Future) SetSettlementPrice(ctx context.Context, priceName string, settlementPrice uint64) {
	od := oracles.OracleData{Data: map[string]string{}}
	od.Data[priceName] = strconv.FormatUint(settlementPrice, 10)
	f.updateSettlementPrice(ctx, od)
}
