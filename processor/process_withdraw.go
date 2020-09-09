package processor

import (
	"context"
	"errors"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
)

var (
	ErrMissingWithdrawERC20Ext = errors.New("missing withdraw submission erc20 ext")
)

func (p *Processor) processWithdraw(ctx context.Context, w *types.WithdrawSubmission) error {
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
		ext := w.Ext.GetErc20()
		if ext == nil {
			return ErrMissingWithdrawERC20Ext
		}
		return p.banking.LockWithdrawalERC20(ctx, w.PartyID, w.Asset, w.Amount, ext)
	default:
		return errors.New("unimplemented withdrawal")
	}
}
