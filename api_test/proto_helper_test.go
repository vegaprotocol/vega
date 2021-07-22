package api_test

import (
	proto "code.vegaprotocol.io/data-node/proto/vega"
	"code.vegaprotocol.io/data-node/types"
	"code.vegaprotocol.io/data-node/types/num"
)

func FeeFromProto(f *proto.Fee) *types.Fee {
	return &types.Fee{
		MakerFee:          num.NewUint(f.MakerFee),
		InfrastructureFee: num.NewUint(f.InfrastructureFee),
		LiquidityFee:      num.NewUint(f.LiquidityFee),
	}
}

func TradeFromProto(t *proto.Trade) *types.Trade {
	return &types.Trade{
		Id:                 t.Id,
		MarketId:           t.MarketId,
		Price:              num.NewUint(t.Price),
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
		Amount:      num.NewUint(l.Amount),
		Reference:   l.Reference,
		Type:        l.Type,
		Timestamp:   l.Timestamp,
	}
}

func AccountFromProto(a *proto.Account) *types.Account {
	return &types.Account{
		Id:       a.Id,
		Owner:    a.Owner,
		Balance:  num.NewUint(a.Balance),
		Asset:    a.Asset,
		MarketId: a.MarketId,
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
		Balance: num.NewUint(t.Balance),
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
