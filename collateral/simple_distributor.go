package collateral

import (
	"context"
	"math"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

type request struct {
	amount  float64
	request *types.Transfer
}

type simpleDistributor struct {
	log             *logging.Logger
	marketID        string
	expectCollected *num.Uint
	collected       *num.Uint
	requests        []request
	ts              int64
}

func (s *simpleDistributor) LossSocializationEnabled() bool {
	return s.collected.LT(s.expectCollected)
}

func (s *simpleDistributor) Add(req *types.Transfer) {
	s.requests = append(s.requests, request{
		amount:  float64(num.NewUint(0).Mul(req.Amount.Amount, num.NewUint(0).Div(s.collected, s.expectCollected)).Uint64()),
		request: req,
	})
}

func (s *simpleDistributor) Run(ctx context.Context) []events.Event {
	if s.expectCollected == s.collected {
		return nil
	}

	var (
		totalamount = num.NewUint(0)
		evts        = make([]events.Event, 0, len(s.requests))
		evt         *events.LossSoc
	)
	for _, v := range s.requests {
		totalamount = num.NewUint(0).Add(totalamount, num.NewUint(uint64(math.Floor(v.amount))))
		evt = events.NewLossSocializationEvent(ctx, v.request.Owner, s.marketID, int64(math.Floor(v.amount))-int64(v.request.Amount.Amount.Uint64()), s.ts)
		v.request.Amount.Amount = num.NewUint(uint64(math.Floor(v.amount)))
		s.log.Warn("loss socialization missing funds to be distributed",
			logging.String("party-id", evt.PartyID()),
			logging.Int64("amount", evt.AmountLost()),
			logging.String("market-id", evt.MarketID()))
		evts = append(evts, evt)
	}

	// TODO(): just rounding the stuff, needs to be done differently later
	if totalamount != s.collected {
		// last one get the remaining bits
		mismatch := num.NewUint(0).Sub(s.collected, totalamount)
		s.requests[len(s.requests)-1].request.Amount.Amount = num.NewUint(0).Add(s.requests[len(s.requests)-1].request.Amount.Amount, mismatch)
		evts[len(evts)-1] = events.NewLossSocializationEvent(
			evt.Context(),
			evt.PartyID(),
			evt.MarketID(),
			evt.AmountLost()+int64(mismatch.Uint64()),
			s.ts)
	}

	return evts
}

type lossSocializationEvt struct {
	market     string
	party      string
	amountLost *num.Uint
}

func (e *lossSocializationEvt) MarketID() string {
	return e.market
}

func (e *lossSocializationEvt) PartyID() string {
	return e.party
}

func (e *lossSocializationEvt) AmountLost() *num.Uint {
	return e.amountLost
}
