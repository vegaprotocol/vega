// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package collateral

import (
	"context"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
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
	lType           types.LossType
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
		total  = num.UintZero()
		evts   = make([]events.Event, 0, len(s.requests))
		evt    *events.LossSoc
		netReq *request
	)
	for _, v := range s.requests {
		total.AddSum(v.amt)
		loss, _ := num.UintZero().Delta(v.amt, v.request.Amount.Amount)
		v.request.Amount.Amount = v.amt.Clone()
		if v.request.Owner == types.NetworkParty {
			v := v
			netReq = &v
		}
		evt = events.NewLossSocializationEvent(ctx, v.request.Owner, s.marketID, loss, true, s.ts, s.lType)
		s.log.Warn("loss socialization missing funds to be distributed",
			logging.String("party-id", evt.PartyID()),
			logging.BigInt("amount", evt.Amount()),
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
		// if the remainder > the loss amount, this rounding error was profitable
		// so the loss socialisation event should be flagged as profit
		// profit will be true if the shortfall < mismatch amount
		loss, profit := mismatch.Delta(evt.Amount().U, mismatch)
		evts[len(evts)-1] = events.NewLossSocializationEvent(
			evt.Context(),
			evt.PartyID(),
			evt.MarketID(),
			loss,
			!profit, // true if party still lost out, false if mismatch > shortfall
			s.ts,
			s.lType,
		)
	}
	return evts
}
