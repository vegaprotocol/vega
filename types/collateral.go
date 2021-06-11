//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package types

import "code.vegaprotocol.io/vega/proto"

type Account struct {
	Id       string
	Owner    string
	Balance  uint64
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
		Balance:  a.Balance,
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
	Amount      uint64
	MinAmount   uint64
	Asset       string
	Reference   string
}

func (t *TransferRequest) IntoProto() *proto.TransferRequest {
	return &proto.TransferRequest{
		FromAccount: Accounts(t.FromAccount).IntoProto(),
		ToAccount:   Accounts(t.ToAccount).IntoProto(),
		Amount:      t.Amount,
		MinAmount:   t.MinAmount,
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
	Balance uint64
}

func (t *TransferBalance) IntoProto() *proto.TransferBalance {
	var acc *proto.Account
	if t.Account != nil {
		acc = t.Account.IntoProto()
	}
	return &proto.TransferBalance{
		Account: acc,
		Balance: t.Balance,
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
	Amount      uint64
	Reference   string
	Type        string
	Timestamp   int64
}

func (l *LedgerEntry) IntoProto() *proto.LedgerEntry {
	return &proto.LedgerEntry{
		FromAccount: l.FromAccount,
		ToAccount:   l.ToAccount,
		Amount:      l.Amount,
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
type Transfer = proto.Transfer
type FinancialAmount = proto.FinancialAmount

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

type TransferType = proto.TransferType

const (
	// Default value, always invalid
	TransferType_TRANSFER_TYPE_UNSPECIFIED TransferType = 0
	// Loss
	TransferType_TRANSFER_TYPE_LOSS TransferType = 1
	// Win
	TransferType_TRANSFER_TYPE_WIN TransferType = 2
	// Close
	TransferType_TRANSFER_TYPE_CLOSE TransferType = 3
	// Mark to market loss
	TransferType_TRANSFER_TYPE_MTM_LOSS TransferType = 4
	// Mark to market win
	TransferType_TRANSFER_TYPE_MTM_WIN TransferType = 5
	// Margin too low
	TransferType_TRANSFER_TYPE_MARGIN_LOW TransferType = 6
	// Margin too high
	TransferType_TRANSFER_TYPE_MARGIN_HIGH TransferType = 7
	// Margin was confiscated
	TransferType_TRANSFER_TYPE_MARGIN_CONFISCATED TransferType = 8
	// Pay maker fee
	TransferType_TRANSFER_TYPE_MAKER_FEE_PAY TransferType = 9
	// Receive maker fee
	TransferType_TRANSFER_TYPE_MAKER_FEE_RECEIVE TransferType = 10
	// Pay infrastructure fee
	TransferType_TRANSFER_TYPE_INFRASTRUCTURE_FEE_PAY TransferType = 11
	// Receive infrastructure fee
	TransferType_TRANSFER_TYPE_INFRASTRUCTURE_FEE_DISTRIBUTE TransferType = 12
	// Pay liquidity fee
	TransferType_TRANSFER_TYPE_LIQUIDITY_FEE_PAY TransferType = 13
	// Receive liquidity fee
	TransferType_TRANSFER_TYPE_LIQUIDITY_FEE_DISTRIBUTE TransferType = 14
	// Bond too low
	TransferType_TRANSFER_TYPE_BOND_LOW TransferType = 15
	// Bond too high
	TransferType_TRANSFER_TYPE_BOND_HIGH TransferType = 16
	// Lock amount for withdraw
	TransferType_TRANSFER_TYPE_WITHDRAW_LOCK TransferType = 17
	// Actual withdraw from system
	TransferType_TRANSFER_TYPE_WITHDRAW TransferType = 18
	// Deposit funds
	TransferType_TRANSFER_TYPE_DEPOSIT TransferType = 19
	// Bond slashing
	TransferType_TRANSFER_TYPE_BOND_SLASHING TransferType = 20
)
