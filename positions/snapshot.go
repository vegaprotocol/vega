package positions

import (
	"context"

	"code.vegaprotocol.io/vega/events"

	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/golang/protobuf/proto"
)

type SnapshotEngine struct {
	*Engine
	pl      types.Payload
	hash    []byte
	data    []byte
	changed bool
	buf     *proto.Buffer
}

func NewSnapshotEngine(
	log *logging.Logger, config Config, marketID string) *SnapshotEngine {
	buf := proto.NewBuffer(nil)
	buf.SetDeterministic(true)
	return &SnapshotEngine{
		Engine:  New(log, config, marketID),
		pl:      types.Payload{},
		changed: true,
		buf:     buf,
	}
}

func (e *SnapshotEngine) RegisterOrder(order *types.Order) *MarketPosition {
	e.changed = true
	return e.Engine.RegisterOrder(order)
}

func (e *SnapshotEngine) UnregisterOrder(order *types.Order) *MarketPosition {
	e.changed = true
	return e.Engine.UnregisterOrder(order)
}

func (e *SnapshotEngine) AmendOrder(originalOrder, newOrder *types.Order) *MarketPosition {
	e.changed = true
	return e.Engine.AmendOrder(originalOrder, newOrder)
}

func (e *SnapshotEngine) UpdateNetwork(trade *types.Trade) []events.MarketPosition {
	e.changed = true
	return e.Engine.UpdateNetwork(trade)
}

func (e *SnapshotEngine) Update(trade *types.Trade) []events.MarketPosition {
	e.changed = true
	return e.Engine.Update(trade)
}

func (e *SnapshotEngine) RemoveDistressed(parties []events.MarketPosition) []events.MarketPosition {
	e.changed = true
	return e.Engine.RemoveDistressed(parties)
}

func (e *SnapshotEngine) UpdateMarkPrice(markPrice *num.Uint) []events.MarketPosition {
	e.changed = true
	return e.Engine.UpdateMarkPrice(markPrice)
}

func (e *SnapshotEngine) Namespace() types.SnapshotNamespace {
	return types.PositionsSnapshot
}

func (e *SnapshotEngine) Keys() []string {
	return []string{e.marketID}
}

func (e *SnapshotEngine) GetHash(k string) ([]byte, error) {
	if k != e.marketID {
		return nil, types.ErrSnapshotKeyDoesNotExist
	}

	_, hash, err := e.serialise()
	return hash, err
}

func (e *Engine) GetState(k string) ([]byte, []types.StateProvider, error) {
	if k != e.marketID {
		return nil, nil, types.ErrSnapshotKeyDoesNotExist
	}

	state, _, err := e.serialise()
	return state, nil, err
}

func (e *Engine) LoadState(_ context.Context, payload *types.Payload) ([]types.StateProvider, error) {
	if e.Namespace() != payload.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	switch pl := payload.Data.(type) {
	case *types.PayloadMarketPositions:

		// Check the payload is for this market
		if e.marketID != pl.MarketPositions.MarketID {
			return nil, types.ErrUnknownSnapshotType
		}

		for _, p := range pl.MarketPositions.Positions {
			pos := NewMarketPosition(p.PartyID)
			pos.price = p.Price
			pos.buy = p.Buy
			pos.sell = p.Sell
			pos.size = p.Size
			pos.vwBuyPrice = p.VwBuy
			pos.vwSellPrice = p.VwSell

			e.positionsCpy = append(e.positionsCpy, pos)
			e.positions[p.PartyID] = pos

			e.changed = true
		}
		return nil, nil

	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

// serialise marshal the snapshot state, populating the data and hash fields
// with updated values.
func (e *SnapshotEngine) serialise() ([]byte, []byte, error) {
	if !e.changed {
		return e.data, e.hash, nil // we already have what we need
	}

	positions := make([]*types.MarketPosition, 0, len(e.positionsCpy))

	for _, evt := range e.positionsCpy {
		pos := &types.MarketPosition{
			Price:  evt.Price(),
			Buy:    evt.Buy(),
			Sell:   evt.Sell(),
			Size:   evt.Size(),
			VwBuy:  evt.VWBuy(),
			VwSell: evt.VWSell(),
		}
		positions = append(positions, pos)
	}

	e.pl.Data = &types.PayloadMarketPositions{
		MarketPositions: &types.MarketPositions{
			MarketID:  e.marketID,
			Positions: positions,
		},
	}

	e.buf.Reset()
	err := e.buf.Marshal(e.pl.IntoProto())
	if err != nil {
		return nil, nil, err
	}

	e.data = e.buf.Bytes()
	e.hash = crypto.Hash(e.data)
	e.changed = false

	return e.data, e.hash, nil
}
