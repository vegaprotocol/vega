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

package gql

import (
	"context"

	"code.vegaprotocol.io/vega/datanode/vegatime"
	types "code.vegaprotocol.io/vega/protos/vega"
)

type myWithdrawalResolver VegaResolverRoot

func (r *myWithdrawalResolver) Party(ctx context.Context, obj *types.Withdrawal) (*types.Party, error) {
	return &types.Party{Id: obj.PartyId}, nil
}

func (r *myWithdrawalResolver) Amount(ctx context.Context, obj *types.Withdrawal) (string, error) {
	return obj.Amount, nil
}

func (r *myWithdrawalResolver) Asset(ctx context.Context, obj *types.Withdrawal) (*types.Asset, error) {
	return r.r.getAssetByID(ctx, obj.Asset)
}

func (r *myWithdrawalResolver) TxHash(ctx context.Context, obj *types.Withdrawal) (*string, error) {
	var s *string
	if len(obj.TxHash) > 0 {
		s = &obj.TxHash
	}
	return s, nil
}

func (r *myWithdrawalResolver) CreatedTimestamp(ctx context.Context, obj *types.Withdrawal) (string, error) {
	return vegatime.Format(vegatime.UnixNano(obj.CreatedTimestamp)), nil
}

func (r *myWithdrawalResolver) WithdrawnTimestamp(ctx context.Context, obj *types.Withdrawal) (*string, error) {
	var s *string
	if obj.WithdrawnTimestamp > 0 {
		ts := vegatime.Format(vegatime.UnixNano(obj.WithdrawnTimestamp))
		s = &ts
	}
	return s, nil
}

func (r *myWithdrawalResolver) Details(ctx context.Context, obj *types.Withdrawal) (WithdrawalDetails, error) {
	return withdrawDetailsFromProto(obj.Ext), nil
}

func withdrawDetailsFromProto(w *types.WithdrawExt) WithdrawalDetails {
	if w == nil {
		return nil
	}
	switch ex := w.Ext.(type) {
	case *types.WithdrawExt_Erc20:
		return &Erc20WithdrawalDetails{ReceiverAddress: ex.Erc20.ReceiverAddress}
	default:
		return nil
	}
}
