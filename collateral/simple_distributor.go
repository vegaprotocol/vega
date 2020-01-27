package collateral

import (
	"math"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
)

type request struct {
	amount  float64
	request *types.Transfer
}

type simpleDistributor struct {
	marketID        string
	expectCollected int64
	collected       int64
	requests        []request
}

func (s *simpleDistributor) LossSocializationEnabled() bool {
	return s.collected < s.expectCollected
}

func (s *simpleDistributor) Add(req *types.Transfer) {
	s.requests = append(s.requests, request{
		amount:  float64(req.Amount.Amount*int64(req.Size)) * (float64(s.collected) / float64(s.expectCollected)),
		request: req,
	})
}

func (s *simpleDistributor) Run() []events.LossSocialization {
	if s.expectCollected == s.collected {
		return []events.LossSocialization{}
	}

	var (
		totalamount int64
		evts        = make([]events.LossSocialization, 0, len(s.requests))
		evt         *lossSocializationEvt
	)
	for _, v := range s.requests {
		totalamount += int64(math.Floor(v.amount))
		evt = &lossSocializationEvt{
			market:     s.marketID,
			party:      v.request.Owner,
			amountLost: v.request.Amount.Amount - int64(math.Floor(v.amount)),
		}
		v.request.Amount.Amount = int64(math.Floor(v.amount))
		evts = append(evts, evt)
	}

	// TODO(): just rounding the stuff, needs to be done differently later
	if totalamount != s.collected {
		// last one get the remaining bits
		s.requests[len(s.requests)-1].request.Amount.Amount += s.collected - totalamount
		evt.amountLost -= s.collected - totalamount
	}

	return evts
}

type lossSocializationEvt struct {
	market     string
	party      string
	amountLost int64
}

func (e *lossSocializationEvt) MarketID() string {
	return e.market
}

func (e *lossSocializationEvt) PartyID() string {
	return e.party
}

func (e *lossSocializationEvt) AmountLost() int64 {
	return e.amountLost
}
