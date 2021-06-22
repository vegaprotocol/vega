package collateral

import (
	"context"

	"github.com/shopspring/decimal"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

type request struct {
	amount  num.Decimal
	request *types.Transfer
}

type simpleDistributor struct {
	log             *logging.Logger
	marketID        string
	expectCollected num.Decimal
	collected       num.Decimal
	requests        []request
	ts              int64
}

func (s *simpleDistributor) LossSocializationEnabled() bool {
	return s.collected.LessThan(s.expectCollected)
}

func (s *simpleDistributor) Add(req *types.Transfer) {
	amount := decimal.RequireFromString(req.Amount.Amount.String()).Mul(s.collected.Div(s.expectCollected))
	s.requests = append(s.requests, request{
		amount:  amount,
		request: req,
	})
}

func (s *simpleDistributor) Run(ctx context.Context) []events.Event {
	if s.expectCollected == s.collected {
		return nil
	}

	var (
		totalamount num.Decimal
		evts        = make([]events.Event, 0, len(s.requests))
		evt         *events.LossSoc
	)
	for _, v := range s.requests {
		totalamount = totalamount.Add(v.amount.Floor())
		loss := v.amount.Floor().Sub(decimal.RequireFromString(v.request.Amount.Amount.String()))
		evt = events.NewLossSocializationEvent(ctx, v.request.Owner, s.marketID, &loss, nil, s.ts)
		v.request.Amount.Amount, _ = num.UintFromBig(v.amount.Floor().BigInt())
		s.log.Warn("loss socialization missing funds to be distributed",
			logging.String("party-id", evt.PartyID()),
			logging.String("amount", evt.Loss().String()),
			logging.String("market-id", evt.MarketID()))
		evts = append(evts, evt)
	}

	// TODO(): just rounding the stuff, needs to be done differently later
	if totalamount != decimal.RequireFromString(s.collected.String()) {
		// last one get the remaining bits
		mismatch := s.collected.Sub(totalamount)
		bigIntMismatch, _ := num.UintFromBig(mismatch.BigInt())
		s.requests[len(s.requests)-1].request.Amount.Amount = num.NewUint(0).Add(s.requests[len(s.requests)-1].request.Amount.Amount, bigIntMismatch)
		loss := evt.Loss().Add(mismatch).Round(0)
		evts[len(evts)-1] = events.NewLossSocializationEvent(
			evt.Context(),
			evt.PartyID(),
			evt.MarketID(),
			&loss,
			nil,
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
