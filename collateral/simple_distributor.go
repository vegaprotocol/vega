package collateral

import (
	"math"

	types "code.vegaprotocol.io/vega/proto"
)

type request struct {
	amount  float64
	request *types.Transfer
}

type simpleDistributor struct {
	expectCollected int64
	collected       int64
	requests        []request
}

func (s *simpleDistributor) Add(req *types.Transfer) {
	s.requests = append(s.requests, request{
		amount:  float64(req.Amount.Amount*int64(req.Size)) * (float64(s.collected) / float64(s.expectCollected)),
		request: req,
	})

}

func (s *simpleDistributor) Run() {
	if s.expectCollected == s.collected {
		return
	}
	var totalamount int64
	for _, v := range s.requests {
		totalamount += int64(math.Floor(v.amount))
		v.request.Amount.Amount = int64(math.Floor(v.amount))
	}

	// TODO(): just rounding the stuff, needs to be done differently later
	if totalamount != s.collected {
		s.requests[0].request.Amount.Amount += s.collected - totalamount
	}

}
