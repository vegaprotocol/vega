package positions

import (
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/protobuf/proto"
)

type positionsSnapshotState struct {
	pl      types.Payload
	hash    []byte
	data    []byte
	changed bool
}

// serialise marshal the snapshot state, populating the data and hash fields
// with updated values.
func (e *Engine) serialise() ([]byte, []byte, error) {
	if !e.pss.changed {
		return e.pss.data, e.pss.hash, nil // we already have what we need
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

	e.pss.pl.Data = &types.PayloadMarketPositions{
		MarketPositions: &types.MarketPositions{
			MarketID:  e.marketID,
			Positions: positions,
		},
	}

	data, err := proto.Marshal(e.pss.pl.IntoProto())
	if err != nil {
		return nil, nil, err
	}

	e.pss.data = data
	e.pss.hash = crypto.Hash(data)
	e.pss.changed = false

	return e.pss.data, e.pss.hash, nil
}

func (e *Engine) Namespace() types.SnapshotNamespace {
	return types.PositionsSnapshot
}

func (e *Engine) Keys() []string {
	return []string{e.marketID}
}

func (e *Engine) GetHash(k string) ([]byte, error) {
	if k != e.marketID {
		return nil, types.ErrSnapshotKeyDoesNotExist
	}

	_, hash, err := e.serialise()
	return hash, err
}

func (e *Engine) GetState(k string) ([]byte, error) {
	if k != e.marketID {
		return nil, types.ErrSnapshotKeyDoesNotExist
	}

	state, _, err := e.serialise()
	return state, err
}

func (e *Engine) Snapshot() (map[string][]byte, error) {
	state, _, err := e.serialise()
	return map[string][]byte{e.marketID: state}, err
}

func (e *Engine) LoadState(payload *types.Payload) error {
	if e.Namespace() != payload.Data.Namespace() {
		return types.ErrInvalidSnapshotNamespace
	}

	switch pl := payload.Data.(type) {
	case *types.PayloadMarketPositions:

		// Check the payload is for this market
		if e.marketID != pl.MarketPositions.MarketID {
			return types.ErrUnknownSnapshotType
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

			e.pss.changed = true
		}
		return nil

	default:
		return types.ErrUnknownSnapshotType
	}
}
