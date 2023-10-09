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

package processor

import (
	"context"
	"errors"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"
)

var ErrMissingWithdrawERC20Ext = errors.New("missing withdraw submission erc20 ext")

func (app *App) processWithdraw(ctx context.Context, w *types.WithdrawSubmission, id string, party string) (err error) {
	asset, err := app.assets.Get(w.Asset)
	if err != nil {
		app.log.Error("invalid vega asset ID for withdrawal",
			logging.Error(err),
			logging.BigUint("amount", w.Amount),
			logging.AssetID(w.Asset))
		return err
	}

	switch {
	case asset.IsBuiltinAsset():
		return app.banking.WithdrawBuiltinAsset(ctx, id, party, w.Asset, w.Amount)
	case asset.IsERC20():
		if w.Ext == nil {
			return ErrMissingWithdrawERC20Ext
		}
		ext := w.Ext.GetErc20()
		if ext == nil {
			return ErrMissingWithdrawERC20Ext
		}
		return app.banking.WithdrawERC20(ctx, id, party, w.Asset, w.Amount, ext.Erc20)
	}

	return errors.New("unimplemented withdrawal")
}
