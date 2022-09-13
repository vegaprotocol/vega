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

package types

import (
	"fmt"

	"code.vegaprotocol.io/vega/libs/num"
	proto "code.vegaprotocol.io/vega/protos/vega"
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
	return fmt.Sprintf(
		"ID(%s) owner(%s) balance(%s) asset(%s) marketID(%s) type(%s)",
		a.ID,
		a.Owner,
		uintPointerToString(a.Balance),
		a.Asset,
		a.MarketID,
		a.Type.String(),
	)
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

type TransferInstructionRequest struct {
	FromAccount []*Account
	ToAccount   []*Account
	Amount      *num.Uint
	MinAmount   *num.Uint
	Asset       string
	Reference   string
}

func (t *TransferInstructionRequest) IntoProto() *proto.TransferInstructionRequest {
	return &proto.TransferInstructionRequest{
		FromAccount: Accounts(t.FromAccount).IntoProto(),
		ToAccount:   Accounts(t.ToAccount).IntoProto(),
		Amount:      num.UintToString(t.Amount),
		MinAmount:   num.UintToString(t.MinAmount),
		Asset:       t.Asset,
		Reference:   t.Reference,
	}
}

type TransferInstructionResponse struct {
	TransferInstructions []*LedgerEntry
	Balances             []*TransferInstructionBalance
}

func (t *TransferInstructionResponse) IntoProto() *proto.TransferInstructionResponse {
	return &proto.TransferInstructionResponse{
		Transfers: LedgerEntries(t.TransferInstructions).IntoProto(),
		Balances:  TransferInstructionBalances(t.Balances).IntoProto(),
	}
}

type TransferInstructionResponses []*TransferInstructionResponse

func (a TransferInstructionResponses) IntoProto() []*proto.TransferInstructionResponse {
	out := make([]*proto.TransferInstructionResponse, 0, len(a))
	for _, v := range a {
		out = append(out, v.IntoProto())
	}
	return out
}

type TransferInstructionBalance struct {
	Account *Account
	Balance *num.Uint
}

func (t *TransferInstructionBalance) IntoProto() *proto.TransferInstructionBalance {
	var acc *proto.Account
	if t.Account != nil {
		acc = t.Account.IntoProto()
	}
	return &proto.TransferInstructionBalance{
		Account: acc,
		Balance: t.Balance.String(),
	}
}

type TransferInstructionBalances []*TransferInstructionBalance

func (a TransferInstructionBalances) IntoProto() []*proto.TransferInstructionBalance {
	out := make([]*proto.TransferInstructionBalance, 0, len(a))
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
	// This account is created to maintain liquidity providers funds commitments.
	AccountTypeBond AccountType = proto.AccountType_ACCOUNT_TYPE_BOND
	// External account represents an external source (deposit/withdrawal).
	AccountTypeExternal AccountType = proto.AccountType_ACCOUNT_TYPE_EXTERNAL
	// Global reward accounts contain rewards per asset.
	AccountTypeGlobalReward AccountType = proto.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD
	// Global account to hold pending transfers.
	AccountTypePendingTransfers AccountType = proto.AccountType_ACCOUNT_TYPE_PENDING_TRANSFER_INSTRUCTIONS
	// Asset account for paid taker fees.
	AccountTypeMakerPaidFeeReward AccountType = proto.AccountType_ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES
	// Asset account for received maker fees.
	AccountTypeMakerReceivedFeeReward AccountType = proto.AccountType_ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES
	// Asset account for received LP fees.
	AccountTypeLPFeeReward AccountType = proto.AccountType_ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES
	// Asset account for market proposers.
	AccountTypeMarketProposerReward AccountType = proto.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS
)
