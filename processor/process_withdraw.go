package processor

import (
	"context"
	"errors"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto/gen/golang"
)

var (
	ErrMissingWithdrawERC20Ext = errors.New("missing withdraw submission erc20 ext")
)

func (app *App) processWithdraw(ctx context.Context, w *types.WithdrawSubmission, id string) error {
	asset, err := app.assets.Get(w.Asset)
	if err != nil {
		app.log.Error("invalid vega asset ID for withdrawal",
			logging.Error(err),
			logging.String("party-id", w.PartyID),
			logging.Uint64("amount", w.Amount),
			logging.String("asset-id", w.Asset))
		return err
	}

	switch {
	case asset.IsBuiltinAsset():
		return app.banking.WithdrawalBuiltinAsset(ctx, id, w.PartyID, w.Asset, w.Amount)
	case asset.IsERC20():
		ext := w.Ext.GetErc20()
		if ext == nil {
			return ErrMissingWithdrawERC20Ext
		}
		return app.banking.LockWithdrawalERC20(ctx, id, w.PartyID, w.Asset, w.Amount, ext)
	}

	return errors.New("unimplemented withdrawal")
}
