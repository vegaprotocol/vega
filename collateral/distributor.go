package collateral

import types "code.vegaprotocol.io/vega/proto"

type distributor struct {
	// fields used to track running totals + delta
	expLoss, lossDelta uint64
}

func (d *distributor) amountCB(req *types.TransferRequest, isLoss bool) {
	if isLoss {
		d.expLoss += req.Amount
		return
	}
	// if delta isn't set, don't do anything
	if d.lossDelta == 0 || d.expLoss == d.lossDelta {
		return
	}
	req.Amount = uint64(float64(req.Amount) / float64(d.expLoss) * float64(d.lossDelta))
}

func (d *distributor) registerTransfer(res *types.TransferResponse) {
	// lossDelta represents the _actual_ loss taken from the accounts
	for _, acc := range res.Balances {
		d.lossDelta += uint64(acc.Balance)
	}
}
