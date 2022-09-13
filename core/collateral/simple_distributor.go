// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
	request *types.TransferInstruction
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

func (s *simpleDistributor) Add(req *types.TransferInstruction) {
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
			continue // network events are to be ignored
		}
		evt = events.NewLossSocializationEvent(ctx, v.request.Owner, s.marketID, loss, true, s.ts)
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
		// decAmt is negative
		loss := mismatch.Sub(evt.Amount().U, mismatch)
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
