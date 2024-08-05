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

package types

import (
	"fmt"

	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/libs/stringer"
	proto "code.vegaprotocol.io/vega/protos/vega"
)

const (
	systemOwner = "*"
	noMarket    = "!"
)

type AccountDetails struct {
	Owner    string
	AssetID  string
	MarketID string
	Type     AccountType
}

func (ad *AccountDetails) ID() string {
	idbuf := make([]byte, 256)
	marketID, partyID := ad.MarketID, ad.Owner
	if len(marketID) <= 0 {
		marketID = noMarket
	}

	// market account
	if len(partyID) <= 0 {
		partyID = systemOwner
	}

	copy(idbuf, marketID)
	ln := len(marketID)
	copy(idbuf[ln:], partyID)
	ln += len(partyID)
	copy(idbuf[ln:], []byte(ad.AssetID))
	ln += len(ad.AssetID)
	idbuf[ln] = byte(ad.Type + 48)
	return string(idbuf[:ln+1])
}

func (ad *AccountDetails) IntoProto() *proto.AccountDetails {
	var marketID, owner *string
	if ad.Owner != systemOwner {
		owner = ptr.From(ad.Owner)
	}
	if ad.MarketID != noMarket {
		marketID = ptr.From(ad.MarketID)
	}

	return &proto.AccountDetails{
		Owner:    owner,
		MarketId: marketID,
		AssetId:  ad.AssetID,
		Type:     ad.Type,
	}
}

type Account struct {
	ID       string
	Owner    string
	Balance  *num.Uint
	Asset    string
	MarketID string // NB: this market may not always refer to a valid market id. instead in the case of transfers it just represents a hash corresponding to a dispatch metric.
	Type     AccountType
}

func (a Account) ToDetails() *AccountDetails {
	return &AccountDetails{
		Owner:    a.Owner,
		MarketID: a.MarketID,
		AssetID:  a.Asset,
		Type:     a.Type,
	}
}

func (a Account) String() string {
	return fmt.Sprintf(
		"ID(%s) owner(%s) balance(%s) asset(%s) marketID(%s) type(%s)",
		a.ID,
		a.Owner,
		stringer.PtrToString(a.Balance),
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

type TransferRequest struct {
	FromAccount []*Account
	ToAccount   []*Account
	Amount      *num.Uint
	MinAmount   *num.Uint
	Asset       string
	// Reference   string
	Type       TransferType
	TransferID *string
}

func (t *TransferRequest) IntoProto() *proto.TransferRequest {
	return &proto.TransferRequest{
		FromAccount: Accounts(t.FromAccount).IntoProto(),
		ToAccount:   Accounts(t.ToAccount).IntoProto(),
		Amount:      num.UintToString(t.Amount),
		MinAmount:   num.UintToString(t.MinAmount),
		Asset:       t.Asset,
		// Reference:   t.Reference,
	}
}

type LedgerMovement struct {
	Entries  []*LedgerEntry
	Balances []*PostTransferBalance
}

func (t *LedgerMovement) IntoProto() *proto.LedgerMovement {
	return &proto.LedgerMovement{
		Entries:  LedgerEntries(t.Entries).IntoProto(),
		Balances: PostTransferBalances(t.Balances).IntoProto(),
	}
}

type LedgerMovements []*LedgerMovement

func (a LedgerMovements) IntoProto() []*proto.LedgerMovement {
	out := make([]*proto.LedgerMovement, 0, len(a))
	for _, v := range a {
		out = append(out, v.IntoProto())
	}
	return out
}

type PostTransferBalance struct {
	Account *Account
	Balance *num.Uint
}

func (t *PostTransferBalance) IntoProto() *proto.PostTransferBalance {
	var acc *proto.AccountDetails
	if t.Account != nil {
		acc = t.Account.ToDetails().IntoProto()
	}
	return &proto.PostTransferBalance{
		Account: acc,
		Balance: t.Balance.String(),
	}
}

type PostTransferBalances []*PostTransferBalance

func (a PostTransferBalances) IntoProto() []*proto.PostTransferBalance {
	out := make([]*proto.PostTransferBalance, 0, len(a))
	for _, v := range a {
		out = append(out, v.IntoProto())
	}
	return out
}

type LedgerEntry struct {
	FromAccount        *AccountDetails
	ToAccount          *AccountDetails
	Amount             *num.Uint
	Type               TransferType
	Timestamp          int64
	FromAccountBalance *num.Uint
	ToAccountBalance   *num.Uint
	TransferID         *string
}

func (l *LedgerEntry) IntoProto() *proto.LedgerEntry {
	return &proto.LedgerEntry{
		FromAccount:        l.FromAccount.IntoProto(),
		ToAccount:          l.ToAccount.IntoProto(),
		Amount:             num.UintToString(l.Amount),
		Type:               l.Type,
		Timestamp:          l.Timestamp,
		FromAccountBalance: num.UintToString(l.FromAccountBalance),
		ToAccountBalance:   num.UintToString(l.ToAccountBalance),
		TransferId:         l.TransferID,
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

type AccountType = proto.AccountType

const (
	// Default value.
	AccountTypeUnspecified AccountType = proto.AccountType_ACCOUNT_TYPE_UNSPECIFIED
	// Per asset network treasury.
	AccountTypeNetworkTreasury AccountType = proto.AccountType_ACCOUNT_TYPE_NETWORK_TREASURY
	// Insurance pool accounts contain insurance pool funds for a market.
	AccountTypeInsurance AccountType = proto.AccountType_ACCOUNT_TYPE_INSURANCE
	// Settlement accounts exist only during settlement or mark-to-market.
	AccountTypeSettlement AccountType = proto.AccountType_ACCOUNT_TYPE_SETTLEMENT
	// Global insurance account for the asset.
	AccountTypeGlobalInsurance AccountType = proto.AccountType_ACCOUNT_TYPE_GLOBAL_INSURANCE

	// Margin accounts contain margin funds for a party and each party will
	// have multiple margin accounts, one for each market they have traded in
	//
	// Margin account funds will alter as margin requirements on positions change.
	AccountTypeMargin AccountType = proto.AccountType_ACCOUNT_TYPE_MARGIN

	// Margin account for isolated margin mode.
	AccountTypeOrderMargin AccountType = proto.AccountType_ACCOUNT_TYPE_ORDER_MARGIN

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
	AccountTypePendingTransfers AccountType = proto.AccountType_ACCOUNT_TYPE_PENDING_TRANSFERS
	// Asset account for paid taker fees.
	AccountTypeMakerPaidFeeReward AccountType = proto.AccountType_ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES
	// Asset account for received maker fees.
	AccountTypeMakerReceivedFeeReward AccountType = proto.AccountType_ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES
	// Asset account for received LP fees.
	AccountTypeLPFeeReward AccountType = proto.AccountType_ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES
	// Asset account for market proposers.
	AccountTypeMarketProposerReward AccountType = proto.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS
	// Market account for holding in-flight spot passive orders funds.
	AccountTypeHolding AccountType = proto.AccountType_ACCOUNT_TYPE_HOLDING
	// Market account per LP to receive liquidity fees.
	AccountTypeLPLiquidityFees                AccountType = proto.AccountType_ACCOUNT_TYPE_LP_LIQUIDITY_FEES
	AccountTypeLiquidityFeesBonusDistribution AccountType = proto.AccountType_ACCOUNT_TYPE_LIQUIDITY_FEES_BONUS_DISTRIBUTION
	AccountTypeVestingRewards                 AccountType = proto.AccountType_ACCOUNT_TYPE_VESTING_REWARDS
	AccountTypeVestedRewards                  AccountType = proto.AccountType_ACCOUNT_TYPE_VESTED_REWARDS
	// Reward account for average notional metric.
	AccountTypeAverageNotionalReward AccountType = proto.AccountType_ACCOUNT_TYPE_REWARD_AVERAGE_NOTIONAL
	// Reward account for relative return metric.
	AccountTypeRelativeReturnReward AccountType = proto.AccountType_ACCOUNT_TYPE_REWARD_RELATIVE_RETURN
	// Reward account for return volatility metric.
	AccountTypeReturnVolatilityReward AccountType = proto.AccountType_ACCOUNT_TYPE_REWARD_RETURN_VOLATILITY
	// Reward account for validator ranking metric.
	AccountTypeValidatorRankingReward AccountType = proto.AccountType_ACCOUNT_TYPE_REWARD_VALIDATOR_RANKING
	// Account for pending fee referral rewards.
	AccountTypePendingFeeReferralReward AccountType = proto.AccountType_ACCOUNT_TYPE_PENDING_FEE_REFERRAL_REWARD
	// Account for realised return rewards.
	AccountTypeRealisedReturnReward = proto.AccountType_ACCOUNT_TYPE_REWARD_REALISED_RETURN
)
