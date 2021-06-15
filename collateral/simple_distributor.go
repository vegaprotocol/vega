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
	amount := num.DecimalFromUint(req.Amount.Amount).Mul(s.collected.Div(s.expectCollected))
	s.requests = append(s.requests, request{
		amount:  amount,
		request: req,
	})
}

func (s *simpleDistributor) Run(ctx context.Context) []events.Event {
	if s.expectCollected.Equal(s.collected) {
		return nil
	}

	var (
		totalamount num.Decimal
		evts        = make([]events.Event, 0, len(s.requests))
		evt         *events.LossSoc
	)
	for _, v := range s.requests {
		amt := v.amount.Floor()
		totalamount = totalamount.Add(amt)
		loss := amt.Sub(num.DecimalFromUint(v.request.Amount.Amount))
		bigIntLoss, _ := num.UintFromBig(loss.BigInt())
		evt = events.NewLossSocializationEvent(ctx, v.request.Owner, s.marketID, bigIntLoss, true, s.ts)
		v.request.Amount.Amount, _ = num.UintFromBig(amt.BigInt())
		s.log.Warn("loss socialization missing funds to be distributed",
			logging.String("party-id", evt.PartyID()),
			logging.Int64("amount", evt.AmountLost()),
			logging.String("market-id", evt.MarketID()))
		evts = append(evts, evt)
	}

	// TODO(): just rounding the stuff, needs to be done differently later
	if !totalamount.Equal(s.collected) {
		// last one get the remaining bits
		mismatch := s.collected.Sub(totalamount)
		bigIntMismatch, _ := num.UintFromBig(mismatch.BigInt())
		s.requests[len(s.requests)-1].request.Amount.Amount.AddSum(bigIntMismatch)
		// decAmt is negative
		decAmt := num.DecimalFromUint(evt.AmountUint()).Sub(mismatch).Round(0)
		loss, _ := num.UintFromBig(decAmt.BigInt())
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
	return e.amountLost
}
