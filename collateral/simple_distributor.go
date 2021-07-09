package collateral

import (
	"context"

	"code.vegaprotocol.io/data-node/events"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/types"
	"code.vegaprotocol.io/data-node/types/num"
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
		total  = num.Zero()
		evts   = make([]events.Event, 0, len(s.requests))
		evt    *events.LossSoc
		netReq *request
	)
	for _, v := range s.requests {
		total.AddSum(v.amt)
		loss, _ := num.Zero().Delta(v.amt, v.request.Amount.Amount)
		v.request.Amount.Amount = v.amt.Clone()
		if v.request.Owner == types.NetworkParty {
			netReq = &v
			continue // network events are to be ignored
		}
		evt = events.NewLossSocializationEvent(ctx, v.request.Owner, s.marketID, loss, true, s.ts)
		s.log.Warn("loss socialization missing funds to be distributed",
			logging.String("party-id", evt.PartyID()),
			logging.Int64("amount", evt.AmountLost()),
			logging.String("market-id", evt.MarketID()))
		evts = append(evts, evt)
	}

	if total.NEQ(s.collected) {
		mismatch, _ := total.Delta(s.collected, total)
		if netReq != nil {
			netReq.request.Amount.Amount.AddSum(mismatch)
			return evts
		}
		// last one get the remaining bits
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
