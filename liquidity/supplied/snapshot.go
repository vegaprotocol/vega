package supplied

import (
	"sort"

	snapshotpb "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/types/num"
)

func (e *Engine) mapToSlice(m map[num.Uint]num.Decimal) []*snapshotpb.LiquidityPriceProbabilityPair {
	slice := make([]*snapshotpb.LiquidityPriceProbabilityPair, 0, len(m))
	for k, v := range m {
		slice = append(slice, &snapshotpb.LiquidityPriceProbabilityPair{Price: k.String(), Probability: v.String()})
	}

	sort.SliceStable(slice, func(i, j int) bool { return slice[i].Price < slice[j].Price })
	return slice
}

func (e *Engine) sliceToMap(lppp []*snapshotpb.LiquidityPriceProbabilityPair) map[num.Uint]num.Decimal {
	m := make(map[num.Uint]num.Decimal, len(lppp))
	for _, pp := range lppp {
		price, _ := num.UintFromString(pp.Price, 10)
		probability, _ := num.DecimalFromString(pp.Probability)
		m[*price] = probability
	}
	return m
}

func (e *Engine) HasUpdates() bool {
	return e.changed
}

func (e *Engine) ResetUpdated() {
	e.changed = false
}

func (e *Engine) Payload() *snapshotpb.Payload {
	// TODO if and when this PR gets approved, this whole snapshot can be deleted
	return &snapshotpb.Payload{
		Data: &snapshotpb.Payload_LiquiditySupplied{
			LiquiditySupplied: &snapshotpb.LiquiditySupplied{
				MarketId: e.marketID,
			},
		},
	}
}

func (e *Engine) Reload(ls *snapshotpb.LiquiditySupplied) error {
	e.changed = true
	return nil
}
