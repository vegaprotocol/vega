package positions

import (
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/protobuf/proto"
)

// shadow returns the positions into the snapshot-version of a position
func (p MarketPosition) shadow() *types.PPosition {
	return &types.PPosition{
		PartyID: p.partyID,
		PSize:   p.size,
		PBuy:    p.buy,
		PSell:   p.sell,
		PPrice:  p.price,
		VwBuy:   p.vwBuyPrice,
		VwSell:  p.vwSellPrice,
	}
}

// serialise
func (e *Engine) serialise() error {

	if !e.changed {
		return nil // we already have what we need
	}

	e.mp.Positions = make([]*types.PPosition, 0, len(e.positionsCpy))
	for _, p := range e.positionsCpy {
		pp := p.(*types.PPosition)
		e.mp.Positions = append(e.mp.Positions, pp)
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
	return []string{e.mp.MarketID}
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

			pos.buy = p.PBuy
			pos.sell = p.PSell
			pos.vwBuyPrice = p.VwBuy
			pos.vwSellPrice = p.VwSell
			pos.price = p.PPrice
			pos.size = p.PSize

			e.positions[p.PartyID] = pos
			e.partyIDToIndex[p.PartyID] = len(e.positionsCpy)
			e.positionsCpy = append(e.positionsCpy, pos.shadow())
		}

		e.changed = true
		return nil

	default:
		return types.ErrUnknownSnapshotType
	}
}
