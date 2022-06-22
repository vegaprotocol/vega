// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package gql

import (
	"context"
	"time"

	"code.vegaprotocol.io/data-node/vegatime"
	types "code.vegaprotocol.io/protos/vega"
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

func (r *myWithdrawalResolver) Status(ctx context.Context, obj *types.Withdrawal) (WithdrawalStatus, error) {
	return convertWithdrawalStatusFromProto(obj.Status)
}

func (r *myWithdrawalResolver) Expiry(ctx context.Context, obj *types.Withdrawal) (string, error) {
	// this is a unix time stamp / non-nano
	return vegatime.Format(time.Unix(obj.Expiry, 0)), nil
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
