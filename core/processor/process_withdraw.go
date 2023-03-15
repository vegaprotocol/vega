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
