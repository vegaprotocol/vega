package positions

import (
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/protobuf/proto"
)

// shadow returns the position converted into the snapshot-type of a position
func (p MarketPosition) shadow() *types.MarketPosition {
	return &types.MarketPosition{
		PartyID: p.partyID,
		Size:    p.size,
		Buy:     p.buy,
		Sell:    p.sell,
		Price:   p.price,
		VwBuy:   p.vwBuyPrice,
		VwSell:  p.vwSellPrice,
	}
}

type positionsSnapshotState struct {
	mp             *types.MarketPositions
	pl             types.Payload
	hash           []byte
	data           []byte
	changed        bool
	partyIDToIndex map[string]int
}

// remove the snapshot state for the position with the given partyID
func (s *positionsSnapshotState) remove(partyID string) {

	i, found := s.partyIDToIndex[partyID]
	if !found {
		return // nothing to remove
	}

	// remove from slice, and index map
	s.mp.Positions = append(s.mp.Positions[:i], s.mp.Positions[i+1:]...)
	delete(s.partyIDToIndex, partyID)

	// all maps to indices > i need to be reduced by one
	for pID, index := range s.partyIDToIndex {
		if index > i {
			s.partyIDToIndex[pID] = index - 1
		}
	}

	s.changed = true
}

// update the snapshot snap with the given mark position
func (s *positionsSnapshotState) update(p *MarketPosition) {

	if _, ok := s.partyIDToIndex[p.partyID]; !ok {
		s.partyIDToIndex[p.partyID] = len(s.mp.Positions)
		s.mp.Positions = append(s.mp.Positions, nil)
	}

	s.mp.Positions[s.partyIDToIndex[p.partyID]] = p.shadow()
	s.changed = true
}

// serialise marshal the snapshot state, populating the data and hash fields
// with updated values
func (s *positionsSnapshotState) serialise() ([]byte, []byte, error) {

	if !s.changed {
		return s.data, s.hash, nil // we already have what we need
	}

	data, err := proto.Marshal(s.pl.IntoProto())
	if err != nil {
		return nil, nil, err
	}

	s.data = data
	s.hash = crypto.Hash(data)
	s.changed = false

	return s.data, s.hash, nil
}

func (e *Engine) Namespace() types.SnapshotNamespace {
	return types.PositionsSnapshot
}

func (e *Engine) Keys() []string {
	return []string{e.pss.pl.Key()}
}

func (e *Engine) GetHash(k string) ([]byte, error) {

	if k != e.pss.pl.Key() {
		return nil, types.ErrSnapshotKeyDoesNotExist
	}

	_, hash, err := e.pss.serialise()
	return hash, err
}

func (e *Engine) GetState(k string) ([]byte, error) {

	if k != e.pss.pl.Key() {
		return nil, types.ErrSnapshotKeyDoesNotExist
	}

	state, _, err := e.pss.serialise()
	return state, err
}

func (e *Engine) Snapshot() (map[string][]byte, error) {

	state, _, err := e.pss.serialise()
	return map[string][]byte{e.pss.mp.MarketID: state}, err
}

func (e *Engine) LoadState(payload *types.Payload) error {

	if e.Namespace() != payload.Data.Namespace() {
		return types.ErrInvalidSnapshotNamespace
	}

	switch pl := payload.Data.(type) {
	case *types.PayloadMarketPositions:

		// Check the payload is for this market
		if e.pss.mp.MarketID != pl.MarketPositions.MarketID {
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

			e.pss.update(pos)
		}
		return nil

	default:
		return types.ErrUnknownSnapshotType
	}
}
