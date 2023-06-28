package common

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/events"
	lmon "code.vegaprotocol.io/vega/core/monitor/liquidity"
	"code.vegaprotocol.io/vega/core/monitor/price"
	"code.vegaprotocol.io/vega/core/oracles"
	"code.vegaprotocol.io/vega/core/risk"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/types/statevar"
	"code.vegaprotocol.io/vega/libs/num"
)

var One = num.UintOne()

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/core/execution/common TimeService,Assets,StateVarEngine,Collateral,OracleEngine,EpochEngine,AuctionState

// InitialOrderVersion is set on `Version` field for every new order submission read from the network.
const InitialOrderVersion = 1

// OracleEngine ...
type OracleEngine interface {
	ListensToSigners(oracles.OracleData) bool
	Subscribe(context.Context, oracles.OracleSpec, oracles.OnMatchedOracleData) (oracles.SubscriptionID, oracles.Unsubscriber, error)
	Unsubscribe(context.Context, oracles.SubscriptionID)
}

// PriceMonitor interface to handle price monitoring/auction triggers
// @TODO the interface shouldn't be imported here.
type PriceMonitor interface {
	OnTimeUpdate(now time.Time)
	CheckPrice(ctx context.Context, as price.AuctionState, trades []*types.Trade, persistent bool) bool
	GetCurrentBounds() []*types.PriceMonitoringBounds
	SetMinDuration(d time.Duration)
	GetValidPriceRange() (num.WrappedDecimal, num.WrappedDecimal)
	// Snapshot
	GetState() *types.PriceMonitor
	Changed() bool
	IsBoundFactorsInitialised() bool
	Initialised() bool
	UpdateSettings(risk.Model, *types.PriceMonitoringSettings)
}

// TimeService ...
type TimeService interface {
	GetTimeNow() time.Time
}

// Broker (no longer need to mock this, use the broker/mocks wrapper).
type Broker interface {
	Send(event events.Event)
	SendBatch(events []events.Event)
}

type StateVarEngine interface {
	RegisterStateVariable(asset, market, name string, converter statevar.Converter, startCalculation func(string, statevar.FinaliseCalculation), trigger []statevar.EventType, result func(context.Context, statevar.StateVariableResult) error) error
	UnregisterStateVariable(asset, market string)
	NewEvent(asset, market string, eventType statevar.EventType)
	ReadyForTimeTrigger(asset, mktID string)
}

type Assets interface {
	Get(assetID string) (*assets.Asset, error)
}

type IDGenerator interface {
	NextID() string
}

// AuctionState ...
//
//nolint:interfacebloat
type AuctionState interface {
	price.AuctionState
	lmon.AuctionState
	// are we in auction, and what auction are we in?
	InAuction() bool
	IsOpeningAuction() bool
	IsPriceAuction() bool
	IsLiquidityAuction() bool
	IsFBA() bool
	IsMonitorAuction() bool
	// is it the start/end of an auction
	AuctionStart() bool
	CanLeave() bool
	// when does the auction start/end
	ExpiresAt() *time.Time
	Start() time.Time
	// signal we've started/ended the auction
	AuctionStarted(ctx context.Context, time time.Time) *events.Auction
	AuctionExtended(ctx context.Context, time time.Time) *events.Auction
	ExtendAuction(delta types.AuctionDuration)
	Left(ctx context.Context, now time.Time) *events.Auction
	// get some data
	Mode() types.MarketTradingMode
	Trigger() types.AuctionTrigger
	ExtensionTrigger() types.AuctionTrigger
	// UpdateMinDuration works out whether or not the current auction period (if applicable) should be extended
	UpdateMinDuration(ctx context.Context, d time.Duration) *events.Auction
	// Snapshot
	GetState() *types.AuctionState
	Changed() bool
}

type EpochEngine interface {
	NotifyOnEpoch(f func(context.Context, types.Epoch), r func(context.Context, types.Epoch))
}

type EligibilityChecker interface {
	IsEligibleForProposerBonus(marketID string, volumeTraded *num.Uint) bool
}

//nolint:interfacebloat
type Collateral interface {
	Deposit(ctx context.Context, party, asset string, amount *num.Uint) (*types.LedgerMovement, error)
	Withdraw(ctx context.Context, party, asset string, amount *num.Uint) (*types.LedgerMovement, error)
	EnableAsset(ctx context.Context, asset types.Asset) error
	GetPartyGeneralAccount(party, asset string) (*types.Account, error)
	GetPartyBondAccount(market, partyID, asset string) (*types.Account, error)
	BondUpdate(ctx context.Context, market string, transfer *types.Transfer) (*types.LedgerMovement, error)
	RemoveBondAccount(partyID, marketID, asset string) error
	MarginUpdateOnOrder(ctx context.Context, marketID string, update events.Risk) (*types.LedgerMovement, events.Margin, error)
	GetPartyMargin(pos events.MarketPosition, asset, marketID string) (events.Margin, error)
	GetPartyMarginAccount(market, party, asset string) (*types.Account, error)
	RollbackMarginUpdateOnOrder(ctx context.Context, marketID string, assetID string, transfer *types.Transfer) (*types.LedgerMovement, error)
	GetOrCreatePartyBondAccount(ctx context.Context, partyID, marketID, asset string) (*types.Account, error)
	CreatePartyMarginAccount(ctx context.Context, partyID, marketID, asset string) (string, error)
	FinalSettlement(ctx context.Context, marketID string, transfers []*types.Transfer) ([]*types.LedgerMovement, error)
	ClearMarket(ctx context.Context, mktID, asset string, parties []string, keepInsurance bool) ([]*types.LedgerMovement, error)
	HasGeneralAccount(party, asset string) bool
	ClearPartyMarginAccount(ctx context.Context, party, market, asset string) (*types.LedgerMovement, error)
	CanCoverBond(market, party, asset string, amount *num.Uint) bool
	Hash() []byte
	TransferFeesContinuousTrading(ctx context.Context, marketID string, assetID string, ft events.FeesTransfer) ([]*types.LedgerMovement, error)
	TransferFees(ctx context.Context, marketID string, assetID string, ft events.FeesTransfer) ([]*types.LedgerMovement, error)
	MarginUpdate(ctx context.Context, marketID string, updates []events.Risk) ([]*types.LedgerMovement, []events.Margin, []events.Margin, error)
	MarkToMarket(ctx context.Context, marketID string, transfers []events.Transfer, asset string) ([]events.Margin, []*types.LedgerMovement, error)
	RemoveDistressed(ctx context.Context, parties []events.MarketPosition, marketID, asset string) (*types.LedgerMovement, error)
	GetMarketLiquidityFeeAccount(market, asset string) (*types.Account, error)
	GetAssetQuantum(asset string) (num.Decimal, error)
	GetInsurancePoolBalance(marketID, asset string) (*num.Uint, bool)
	AssetExists(string) bool
	CreateMarketAccounts(context.Context, string, string) (string, string, error)
	SuccessorInsuranceFraction(ctx context.Context, successor, parent, asset string, fraction num.Decimal) *types.LedgerMovement
	ClearInsurancepool(ctx context.Context, marketID string, asset string, clearFees bool) ([]*types.LedgerMovement, error)
}
