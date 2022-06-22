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

package api_test

import (
	"fmt"

	proto "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

func mustUintFromString(s string) *num.Uint {
	// assume "" == 0
	if len(s) <= 0 {
		return num.Zero()
	}
	u, overflow := num.UintFromString(s, 10)
	if overflow {
		panic(fmt.Sprintf("uint to string overflowed: \"%v\"", s))
	}
	return u
}

func FeeFromProto(f *proto.Fee) *types.Fee {
	return &types.Fee{
		MakerFee:          mustUintFromString(f.MakerFee),
		InfrastructureFee: mustUintFromString(f.InfrastructureFee),
		LiquidityFee:      mustUintFromString(f.LiquidityFee),
	}
}

func TradeFromProto(t *proto.Trade) *types.Trade {
	return &types.Trade{
		ID:                 t.Id,
		MarketID:           t.MarketId,
		MarketPrice:        mustUintFromString(t.Price),
		Price:              mustUintFromString(t.Price),
		Size:               t.Size,
		Buyer:              t.Buyer,
		Seller:             t.Seller,
		Aggressor:          t.Aggressor,
		BuyOrder:           t.BuyOrder,
		SellOrder:          t.SellOrder,
		Timestamp:          t.Timestamp,
		Type:               t.Type,
		BuyerFee:           FeeFromProto(t.BuyerFee),
		SellerFee:          FeeFromProto(t.SellerFee),
		BuyerAuctionBatch:  t.BuyerAuctionBatch,
		SellerAuctionBatch: t.SellerAuctionBatch,
	}
}

func LedgerEntryFromProto(l *proto.LedgerEntry) *types.LedgerEntry {
	return &types.LedgerEntry{
		FromAccount: l.FromAccount,
		ToAccount:   l.ToAccount,
		Amount:      mustUintFromString(l.Amount),
		Reference:   l.Reference,
		Type:        l.Type,
		Timestamp:   l.Timestamp,
	}
}

func AccountFromProto(a *proto.Account) *types.Account {
	return &types.Account{
		ID:       a.Id,
		Owner:    a.Owner,
		Balance:  mustUintFromString(a.Balance),
		Asset:    a.Asset,
		MarketID: a.MarketId,
		Type:     a.Type,
	}
}

func TransferBalanceFromProto(t *proto.TransferBalance) *types.TransferBalance {
	var acc *types.Account
	if t.Account != nil {
		acc = AccountFromProto(t.Account)
	}
	return &types.TransferBalance{
		Account: acc,
		Balance: mustUintFromString(t.Balance),
	}
}

func TransferResponseFromProto(t *proto.TransferResponse) *types.TransferResponse {
	out := types.TransferResponse{}
	var ll []*types.LedgerEntry
	for _, v := range t.Transfers {
		ll = append(ll, LedgerEntryFromProto(v))
	}
	out.Transfers = ll
	var bb []*types.TransferBalance
	for _, v := range t.Balances {
		bb = append(bb, TransferBalanceFromProto(v))
	}
	out.Balances = bb
	return &out
}

func TransferResponsesFromProto(tt []*proto.TransferResponse) []*types.TransferResponse {
	out := make([]*types.TransferResponse, 0, len(tt))
	for _, t := range tt {
		out = append(out, TransferResponseFromProto(t))
	}
	return out
}
