package collateral

import (
	"context"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

type request struct {
	amount  num.Decimal
	amt     *num.Uint
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
	col, exp := num.DecimalFromUint(s.collected), num.DecimalFromUint(s.expectCollected)
	amount := num.DecimalFromUint(req.Amount.Amount).Mul(col.Div(exp)).Floor()
	amt, _ := num.UintFromBig(amount.BigInt())
	s.requests = append(s.requests, request{
		amount:  amount,
		amt:     amt,
		request: req,
	})
}

func (s *simpleDistributor) Run(ctx context.Context) []events.Event {
	if s.expectCollected.EQ(s.collected) {
		return nil
	}

	var (
		total = num.NewUint(0)
		evts  = make([]events.Event, 0, len(s.requests))
		evt   *events.LossSoc
	)
	for _, v := range s.requests {
		total.AddSum(v.amt)
		loss, _ := num.NewUint(0).Delta(v.amt, v.request.Amount.Amount)
		evt = events.NewLossSocializationEvent(ctx, v.request.Owner, s.marketID, loss, true, s.ts)
		v.request.Amount.Amount = v.amt.Clone()
		s.log.Warn("loss socialization missing funds to be distributed",
			logging.String("party-id", evt.PartyID()),
			logging.Int64("amount", evt.AmountLost()),
			logging.String("market-id", evt.MarketID()))
		evts = append(evts, evt)
	}

	if total.NEQ(s.collected) {
		// last one get the remaining bits
		mismatch, _ := total.Delta(s.collected, total)
		s.requests[len(s.requests)-1].request.Amount.Amount.AddSum(mismatch)
		// decAmt is negative
		loss := mismatch.Sub(evt.AmountUint(), mismatch)
		evts[len(evts)-1] = events.NewLossSocializationEvent(
			evt.Context(),
			evt.PartyID(),
			evt.MarketID(),
			loss,
			true,
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
	return e.amountLost.Clone()
}
