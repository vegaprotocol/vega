package matching

import types "code.vegaprotocol.io/vega/proto"

// StaticPrice a type that holds price info needed to reprice pegged orders
// this can be re-used so we don't continuously traverse the orderbook to get the same values for each order
type StaticPrice struct {
	bid, ask, midA, midB uint64
}

func (s StaticPrice) Bid() uint64 {
	return s.bid
}

func (s StaticPrice) Ask() uint64 {
	return s.ask
}

func (s *StaticPrice) Mid(side types.Side) uint64 {
	if side == types.Side_SIDE_BUY {
		return s.MidBid()
	}
	return s.MidAsk()
}

// MidBid == buy
func (s *StaticPrice) MidBid() uint64 {
	if s.midB == 0 {
		mid := s.bid + s.ask
		s.midA = mid / 2
		s.midB = (mid + 1) / 2
	}
	return s.midB
}

// MidAsk == mid sell
func (s StaticPrice) MidAsk() uint64 {
	if s.midB == 0 {
		mid := s.bid + s.ask
		s.midA = mid / 2
		s.midB = (mid + 1) / 2
	}
	return s.midA
}
