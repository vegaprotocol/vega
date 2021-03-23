package products

import (
	"context"

	"code.vegaprotocol.io/vega/oracles"
)

func (f *Future) SetSettlementPrice(ctx context.Context, settlementPrice uint64) {
	od := oracles.OracleData{}
	f.updateSettlementPrice(ctx, od)
}
