package processor

import (
	"context"
	"errors"

	"code.vegaprotocol.io/vega/logging"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
)

var (
	ErrMissingWithdrawERC20Ext = errors.New("missing withdraw submission erc20 ext")
)

func (app *App) processWithdraw(ctx context.Context, w *commandspb.WithdrawSubmission, id string, party string) error {
	asset, err := app.assets.Get(w.Asset)
	if err != nil {
		app.log.Error("invalid vega asset ID for withdrawal",
			logging.Error(err),
			logging.Uint64("amount", w.Amount),
			logging.AssetID(w.Asset))
		return err
	}

	switch {
	case asset.IsBuiltinAsset():
		return app.banking.WithdrawalBuiltinAsset(ctx, id, party, w.Asset, w.Amount)
	case asset.IsERC20():
		ext := w.Ext.GetErc20()
		if ext == nil {
			return ErrMissingWithdrawERC20Ext
		}
		return app.banking.LockWithdrawalERC20(ctx, id, party, w.Asset, w.Amount, ext)
	}

	return errors.New("unimplemented withdrawal")
}
