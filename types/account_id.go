package types

import proto "code.vegaprotocol.io/protos/vega"

type AccountId struct {
	ID       string
	Owner    string
	Asset    string
	MarketID string
	Type     AccountType
}

func AccountIdFromAccount(a *Account) AccountId {
	return AccountId{
		ID:       a.ID,
		Owner:    a.Owner,
		Asset:    a.Asset,
		MarketID: a.MarketID,
		Type:     a.Type,
	}
}

func (a *AccountId) IntoProto() *proto.AccountId {
	return &proto.AccountId{
		ID:       a.ID,
		Owner:    a.Owner,
		Asset:    a.Asset,
		MarketId: a.MarketID,
		Type:     a.Type,
	}
}
