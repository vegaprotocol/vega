package processor

import (
	"context"
	"errors"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
)

func (p *Processor) processWithdraw(ctx context.Context, w *types.Withdraw) error {
	asset, err := p.assets.Get(w.Asset)
	if err != nil {
		if err != nil {
			p.log.Error("invalid vega asset ID for withdrawal",
				logging.Error(err),
				logging.String("party-id", w.PartyID),
				logging.Uint64("amount", w.Amount),
				logging.String("asset-id", w.Asset))
			return err
		}
	}
	switch {
	case asset.IsBuiltinAsset():
		return p.banking.WithdrawalBuiltinAsset(ctx, w.PartyID, w.Asset, w.Amount)
	case asset.IsERC20():
		return errors.New("unimplemented withdrawal for ERC20")
	default:
		return errors.New("unimplemented withdrawal")
	}
}
