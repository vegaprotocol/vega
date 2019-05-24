package collateral

import types "code.vegaprotocol.io/vega/proto"

type distributor struct {
	// fields used to track running totals + delta
	expWin, expLoss, lossDelta uint64
}

func (d distributor) amountCB(req *types.TransferRequest) {
	// if delta isn't set, don't do anything
	if d.lossDelta == 0 || d.expLoss == d.lossDelta {
		return
	}
	req.Amount = uint64(float64(req.Amount) / float64(d.expLoss) * float64(d.lossDelta))
}
