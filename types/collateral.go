package types

import (
	proto "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/types/num"
)

type Account struct {
	ID       string
	Owner    string
	Balance  *num.Uint
	Asset    string
	MarketID string
	Type     AccountType
}

func (a Account) String() string {
	return a.IntoProto().String()
}

func (a *Account) Clone() *Account {
	acccpy := *a
	acccpy.Balance = acccpy.Balance.Clone()
	return &acccpy
}

func AccountFromProto(a *proto.Account) *Account {
	bal, _ := num.UintFromString(a.Balance, 10)
	return &Account{
		ID:       a.Id,
		Owner:    a.Owner,
		Balance:  bal,
		Asset:    a.Asset,
		MarketID: a.MarketId,
		Type:     a.Type,
	}
}

func (a *Account) IntoProto() *proto.Account {
	return &proto.Account{
		Id:       a.ID,
		Owner:    a.Owner,
		Balance:  num.UintToString(a.Balance),
		Asset:    a.Asset,
		MarketId: a.MarketID,
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
		Amount:      num.UintToString(t.Amount),
		MinAmount:   num.UintToString(t.MinAmount),
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
		Balance: t.Balance.String(),
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
		Amount:      num.UintToString(l.Amount),
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
	// Default value.
	AccountTypeUnspecified AccountType = proto.AccountType_ACCOUNT_TYPE_UNSPECIFIED
	// Insurance pool accounts contain insurance pool funds for a market.
	AccountTypeInsurance AccountType = proto.AccountType_ACCOUNT_TYPE_INSURANCE
	// Settlement accounts exist only during settlement or mark-to-market.
	AccountTypeSettlement AccountType = proto.AccountType_ACCOUNT_TYPE_SETTLEMENT
	// Margin accounts contain margin funds for a party and each party will
	// have multiple margin accounts, one for each market they have traded in
	//
	// Margin account funds will alter as margin requirements on positions change.
	AccountTypeMargin AccountType = proto.AccountType_ACCOUNT_TYPE_MARGIN
	// General accounts contains general funds for a party. A party will
	// have multiple general accounts, one for each asset they want
	// to trade with
	//
	// General accounts are where funds are initially deposited or withdrawn from,
	// it is also the account where funds are taken to fulfil fees and initial margin requirements.
	AccountTypeGeneral AccountType = proto.AccountType_ACCOUNT_TYPE_GENERAL
	// Infrastructure accounts contain fees earned by providing infrastructure on Vega.
	AccountTypeFeesInfrastructure AccountType = proto.AccountType_ACCOUNT_TYPE_FEES_INFRASTRUCTURE
	// Liquidity accounts contain fees earned by providing liquidity on Vega markets.
	AccountTypeFeesLiquidity AccountType = proto.AccountType_ACCOUNT_TYPE_FEES_LIQUIDITY
	// This account is created to hold fees earned by placing orders that sit on the book
	// and are then matched with an incoming order to create a trade - These fees reward parties
	// who provide the best priced liquidity that actually allows trading to take place.
	AccountTypeFeesMaker AccountType = proto.AccountType_ACCOUNT_TYPE_FEES_MAKER
	// This account is created to lock funds to be withdrawn by parties.
	AccountTypeLockWithdraw AccountType = proto.AccountType_ACCOUNT_TYPE_LOCK_WITHDRAW
	// This account is created to maintain liquidity providers funds commitments.
	AccountTypeBond AccountType = proto.AccountType_ACCOUNT_TYPE_BOND
	// External account represents an external source (deposit/withdrawal).
	AccountTypeExternal AccountType = proto.AccountType_ACCOUNT_TYPE_EXTERNAL
	// Global reward accounts contain rewards per asset.
	AccountTypeGlobalReward AccountType = proto.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD
	// Global account to hold pending transfers.
	AccountTypePendingTransfers AccountType = proto.AccountType_ACCOUNT_TYPE_PENDING_TRANSFERS
	// Asset account for paid taker fees.
	AccountTypeTakerFeeReward AccountType = proto.AccountType_ACCOUNT_TYPE_REWARD_TAKER_PAID_FEES
	// Asset account for received maker fees.
	AccountTypeMakerFeeReward AccountType = proto.AccountType_ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES
	// Asset account for received LP fees.
	AccountTypeLPFeeReward AccountType = proto.AccountType_ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES
	// Asset account for market proposers.
	AccountTypeMarketProposerReward AccountType = proto.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS
)
