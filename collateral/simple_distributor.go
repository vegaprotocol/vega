package collateral

import (
	"context"
	"math"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
)

type request struct {
	amount  float64
	request *types.Transfer
}

type simpleDistributor struct {
	log             *logging.Logger
	marketID        string
	expectCollected uint64
	collected       uint64
	requests        []request
	ts              int64
}

func (s *simpleDistributor) LossSocializationEnabled() bool {
	return s.collected < s.expectCollected
}

func (s *simpleDistributor) Add(req *types.Transfer) {
	s.requests = append(s.requests, request{
		amount:  float64(req.Amount.Amount) * (float64(s.collected) / float64(s.expectCollected)),
		request: req,
	})
}

func (s *simpleDistributor) Run(ctx context.Context) []events.Event {
	if s.expectCollected == s.collected {
		return nil
	}

	var (
		totalamount uint64
		evts        = make([]events.Event, 0, len(s.requests))
		evt         *events.LossSoc
	)
	for _, v := range s.requests {
		totalamount += uint64(math.Floor(v.amount))
		evt = events.NewLossSocializationEvent(ctx, v.request.Owner, s.marketID, int64(math.Floor(v.amount))-int64(v.request.Amount.Amount), s.ts)
		v.request.Amount.Amount = uint64(math.Floor(v.amount))
		s.log.Warn("loss socialization missing funds to be distributed",
			logging.String("party-id", evt.PartyID()),
			logging.Int64("amount", evt.AmountLost()),
			logging.String("market-id", evt.MarketID()))
		evts = append(evts, evt)
	}

	// TODO(): just rounding the stuff, needs to be done differently later
	if totalamount != s.collected {
		// last one get the remaining bits
		mismatch := s.collected - totalamount
		s.requests[len(s.requests)-1].request.Amount.Amount += mismatch
		evts[len(evts)-1] = events.NewLossSocializationEvent(
			evt.Context(),
			evt.PartyID(),
			evt.MarketID(),
			evt.AmountLost()+int64(mismatch),
			s.ts)
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
