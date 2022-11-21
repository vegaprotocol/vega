package settlement

import (
	"context"
	"sort"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
)

type SnapshotEngine struct {
	*Engine
	p       types.Payload
	stopped bool
}

func NewSnapshotEngine(log *logging.Logger, conf Config, product Product, market string, timeService TimeService, broker Broker, positionFactor num.Decimal) *SnapshotEngine {
	return &SnapshotEngine{
		Engine: New(log, conf, product, market, timeService, broker, positionFactor),
	}
}

// StopSnapshots is called when the engines respective market no longer exists. We need to stop
// taking snapshots and communicate to the snapshot engine to remove us as a provider.
func (e *SnapshotEngine) StopSnapshots() {
	e.log.Debug("market has been cleared, stopping snapshot production", logging.String("marketid", e.market))
	e.stopped = true
}

func (e *SnapshotEngine) Stopped() bool {
	return e.stopped
}

func (e *SnapshotEngine) Namespace() types.SnapshotNamespace {
	return types.SettlementSnapshot
}

func (e *SnapshotEngine) Keys() []string {
	return []string{e.market}
}

func (e *SnapshotEngine) GetState(k string) ([]byte, []types.StateProvider, error) {
	if k != e.market {
		return nil, nil, types.ErrSnapshotKeyDoesNotExist
	}
	state, err := e.serialise()
	return state, nil, err
}

func (e *SnapshotEngine) LoadState(_ context.Context, payload *types.Payload) ([]types.StateProvider, error) {
	if e.Namespace() != payload.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	switch pl := payload.Data.(type) {
	case *types.PayloadSettlement:
		data := pl.SettlementState

		// Check the payload is for this market
		if e.market != data.MarketID {
			return nil, types.ErrUnknownSnapshotType
		}
		e.log.Debug("loading settlement snapshot",
			logging.Int("positions", len(data.Positions)),
			logging.Int("trades", len(data.Trades)),
		)
		// restore positions
		for _, p := range data.Positions {
			e.pos[p.PartyID] = &pos{
				MarketPosition: snapWrap{p},
				party:          p.PartyID,
				size:           p.Size,
				price:          p.Price, // should be fine not cloning this value
			}
		}
		// restore trades
		tradeMap := map[string][]*settlementTrade{}
		for _, trade := range data.Trades {
			party := trade.Party
			st := stTypeToInternal(trade)
			ps, ok := tradeMap[party]
			if !ok {
				ps = make([]*settlementTrade, 0, 5) // some buffer
			}
			tradeMap[party] = append(ps, st)
		}
		e.trades = tradeMap
		// we restored state just fine
		return nil, nil
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (e *SnapshotEngine) serialise() ([]byte, error) {
	data := types.SettlementState{
		MarketPositions: &types.MarketPositions{
			MarketID: e.market,
		},
	}
	positions := make([]*types.MarketPosition, 0, len(e.pos))
	for _, p := range e.pos {
		positions = append(positions, &types.MarketPosition{
			PartyID: p.party,
			Size:    p.size,
			Buy:     p.Buy(),
			Sell:    p.Sell(),
			VwBuy:   p.VWBuy(),
			VwSell:  p.VWSell(),
			Price:   p.price, // no need to clone, we're serialising it in this call
		})
	}
	// now sort by party
	sort.SliceStable(positions, func(i, j int) bool {
		return positions[i].PartyID < positions[j].PartyID
	})
	data.MarketPositions.Positions = positions
	// first get all parties that traded
	tradeParties := make([]string, 0, len(e.trades))
	tradeTotal := 0
	// convert to correct type, keep that in a map
	mapped := make(map[string][]*types.SettlementTrade, len(e.trades))
	for k, trades := range e.trades {
		tradeParties = append(tradeParties, k) // slice of parties
		mapped[k] = internalSTToType(trades, k)
		tradeTotal += len(trades) // keep track of the total trades
	}
	// get map keys sorted
	sort.Strings(tradeParties)
	// now do the trades
	trades := make([]*types.SettlementTrade, 0, tradeTotal)
	for _, p := range tradeParties {
		pp := mapped[p]
		// append trades for party
		trades = append(trades, pp...)
	}
	data.Trades = trades

	// now the payload type to serialise:
	payload := types.Payload{
		Data: &types.PayloadSettlement{
			SettlementState: &data,
		},
	}
	ser, err := proto.Marshal(payload.IntoProto())
	if err != nil {
		return nil, err
	}

	return ser, nil
}

func internalSTToType(trades []*settlementTrade, party string) []*types.SettlementTrade {
	ret := make([]*types.SettlementTrade, 0, len(trades))
	for _, t := range trades {
		ret = append(ret, &types.SettlementTrade{
			Price:       t.price,
			MarketPrice: t.marketPrice,
			Size:        t.size,
			NewSize:     t.newSize,
			Party:       party,
		})
	}
	return ret
}

func stTypeToInternal(st *types.SettlementTrade) *settlementTrade {
	return &settlementTrade{
		size:        st.Size,
		newSize:     st.NewSize,
		price:       st.Price,
		marketPrice: st.MarketPrice,
	}
}
