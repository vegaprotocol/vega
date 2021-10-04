package positions

import (
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/protobuf/proto"
)

// shadow returns the position converted into the snapshot-type of a position
func (p MarketPosition) shadow() *types.PPosition {
	return &types.PPosition{
		PartyID: p.partyID,
		Size:    p.size,
		Buy:     p.buy,
		Sell:    p.sell,
		Price:   p.price,
		VwBuy:   p.vwBuyPrice,
		VwSell:  p.vwSellPrice,
	}
}

func (e *Engine) serialise() error {

	if !e.changed {
		return nil // we already have what we need
	}

	data, err := proto.Marshal(e.pl.IntoProto())
	if err != nil {
		return err
	}

	e.data = data
	e.hash = crypto.Hash(data)
	e.changed = false

	return nil
}

func (e *Engine) Namespace() types.SnapshotNamespace {
	return e.pl.Namespace()
}

func (e *Engine) Keys() []string {
	return []string{e.pl.Key()}
}

func (e *Engine) GetHash(k string) ([]byte, error) {

	if k != e.pl.Key() {
		return nil, types.ErrSnapshotKeyDoesNotExist
	}

	e.serialise()
	return e.hash, nil
}

func (e *Engine) Snapshot() (map[string][]byte, error) {
	return map[string][]byte{
		e.mp.MarketID: e.data,
	}, nil
}

func (e *Engine) GetState(k string) ([]byte, error) {

	if k != e.pl.Key() {
		return nil, types.ErrSnapshotKeyDoesNotExist
	}

	e.serialise()
	return e.data, nil
}

func (e *Engine) LoadState(payload *types.Payload) error {

	if e.Namespace() != payload.Data.Namespace() {
		return types.ErrInvalidSnapshotNamespace
	}

	switch pl := payload.Data.(type) {
	case *types.PayloadMarketPositions:

		// Check the payload is for this market
		if e.mp.MarketID != pl.MarketPositions.MarketID {
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

			e.positions[p.PartyID] = pos
			e.positionsCpy = append(e.positionsCpy, pos)

			e.partyIDToIndex[p.PartyID] = len(e.mp.Positions)
			e.mp.Positions = append(e.mp.Positions, pos.shadow())
		}

		e.changed = true
		return nil

	default:
		return types.ErrUnknownSnapshotType
	}
}
