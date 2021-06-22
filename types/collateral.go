//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package types

import (
	"code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/types/num"
)

type Account struct {
	Id       string
	Owner    string
	Balance  *num.Uint
	Asset    string
	MarketId string
	Type     AccountType
}

func (a Account) String() string {
	return a.IntoProto().String()
}

func (a *Account) IntoProto() *proto.Account {
	return &proto.Account{
		Id:       a.Id,
		Owner:    a.Owner,
		Balance:  a.Balance.Uint64(),
		Asset:    a.Asset,
		MarketId: a.MarketId,
		Type:     a.Type,
	}
}

type Accounts []*Account

func (a Accounts) IntoProto() []*proto.Account {
	out := make([]*proto.Account, 0, len(a))
	for _, v := range a {
		out = append(out, v.IntoProto())
	}
	return out
}

type TransferRequest struct {
	FromAccount []*Account
	ToAccount   []*Account
	Amount      *num.Uint
	MinAmount   *num.Uint
	Asset       string
	Reference   string
}

func (t *TransferRequest) IntoProto() *proto.TransferRequest {
	return &proto.TransferRequest{
		FromAccount: Accounts(t.FromAccount).IntoProto(),
		ToAccount:   Accounts(t.ToAccount).IntoProto(),
		Amount:      t.Amount.Uint64(),
		MinAmount:   t.MinAmount.Uint64(),
		Asset:       t.Asset,
		Reference:   t.Reference,
	}
}

type TransferResponse struct {
	Transfers []*LedgerEntry
	Balances  []*TransferBalance
}

func (t *TransferResponse) IntoProto() *proto.TransferResponse {
	return &proto.TransferResponse{
		Transfers: LedgerEntries(t.Transfers).IntoProto(),
		Balances:  TransferBalances(t.Balances).IntoProto(),
	}
}

type TransferResponses []*TransferResponse

func (a TransferResponses) IntoProto() []*proto.TransferResponse {
	out := make([]*proto.TransferResponse, 0, len(a))
	for _, v := range a {
		out = append(out, v.IntoProto())
	}
	return out
}

type TransferBalance struct {
	Account *Account
	Balance *num.Uint
}

func (t *TransferBalance) IntoProto() *proto.TransferBalance {
	var acc *proto.Account
	if t.Account != nil {
		acc = t.Account.IntoProto()
	}
	return &proto.TransferBalance{
		Account: acc,
		Balance: t.Balance.Uint64(),
	}
}

type TransferBalances []*TransferBalance

func (a TransferBalances) IntoProto() []*proto.TransferBalance {
	out := make([]*proto.TransferBalance, 0, len(a))
	for _, v := range a {
		out = append(out, v.IntoProto())
	}
	return out
}

type LedgerEntry struct {
	FromAccount string
	ToAccount   string
	Amount      *num.Uint
	Reference   string
	Type        string
	Timestamp   int64
}

func (l *LedgerEntry) IntoProto() *proto.LedgerEntry {
	return &proto.LedgerEntry{
		FromAccount: l.FromAccount,
		ToAccount:   l.ToAccount,
		Amount:      l.Amount.Uint64(),
		Reference:   l.Reference,
		Type:        l.Type,
		Timestamp:   l.Timestamp,
	}
}

type LedgerEntries []*LedgerEntry

func (a LedgerEntries) IntoProto() []*proto.LedgerEntry {
	out := make([]*proto.LedgerEntry, 0, len(a))
	for _, v := range a {
		out = append(out, v.IntoProto())
	}
	return out
}

type Party = proto.Party

type AccountType = proto.AccountType

const (
	// Default value
	AccountType_ACCOUNT_TYPE_UNSPECIFIED AccountType = 0
	// Insurance pool accounts contain insurance pool funds for a market
	AccountType_ACCOUNT_TYPE_INSURANCE AccountType = 1
	// Settlement accounts exist only during settlement or mark-to-market
	AccountType_ACCOUNT_TYPE_SETTLEMENT AccountType = 2
	// Margin accounts contain margin funds for a party and each party will
	// have multiple margin accounts, one for each market they have traded in
	//
	// Margin account funds will alter as margin requirements on positions change
	AccountType_ACCOUNT_TYPE_MARGIN AccountType = 3
	// General accounts contains general funds for a party. A party will
	// have multiple general accounts, one for each asset they want
	// to trade with
	//
	// General accounts are where funds are initially deposited or withdrawn from,
	// it is also the account where funds are taken to fulfil fees and initial margin requirements
	AccountType_ACCOUNT_TYPE_GENERAL AccountType = 4
	// Infrastructure accounts contain fees earned by providing infrastructure on Vega
	AccountType_ACCOUNT_TYPE_FEES_INFRASTRUCTURE AccountType = 5
	// Liquidity accounts contain fees earned by providing liquidity on Vega markets
	AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY AccountType = 6
	// This account is created to hold fees earned by placing orders that sit on the book
	// and are then matched with an incoming order to create a trade - These fees reward traders
	// who provide the best priced liquidity that actually allows trading to take place
	AccountType_ACCOUNT_TYPE_FEES_MAKER AccountType = 7
	// This account is created to lock funds to be withdrawn by parties
	AccountType_ACCOUNT_TYPE_LOCK_WITHDRAW AccountType = 8
	// This account is created to maintain liquidity providers funds commitments
	AccountType_ACCOUNT_TYPE_BOND AccountType = 9
	// External account represents an external source (deposit/withdrawal)
	AccountType_ACCOUNT_TYPE_EXTERNAL AccountType = 10
)
