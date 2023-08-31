package common

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/core/assets"
	dscommon "code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/datasource/spec"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/liquidity/v2"
	lmon "code.vegaprotocol.io/vega/core/monitor/liquidity"
	"code.vegaprotocol.io/vega/core/monitor/price"
	"code.vegaprotocol.io/vega/core/risk"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/types/statevar"
	"code.vegaprotocol.io/vega/libs/num"
)

var One = num.UintOne()

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/core/execution/common TimeService,Assets,StateVarEngine,Collateral,OracleEngine,EpochEngine,AuctionState,LiquidityEngine,EquityLikeShares,MarketLiquidityEngine,Teams,AccountBalanceChecker

// InitialOrderVersion is set on `Version` field for every new order submission read from the network.
const InitialOrderVersion = 1

// OracleEngine ...
type OracleEngine interface {
	ListensToSigners(dscommon.Data) bool
	Subscribe(context.Context, spec.Spec, spec.OnMatchedData) (spec.SubscriptionID, spec.Unsubscriber, error)
	Unsubscribe(context.Context, spec.SubscriptionID)
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
	ExtendAuctionSuspension(delta types.AuctionDuration)
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
	BondSpotUpdate(ctx context.Context, market string, transfer *types.Transfer) (*types.LedgerMovement, error)
	RemoveBondAccount(partyID, marketID, asset string) error
	MarginUpdateOnOrder(ctx context.Context, marketID string, update events.Risk) (*types.LedgerMovement, events.Margin, error)
	GetPartyMargin(pos events.MarketPosition, asset, marketID string) (events.Margin, error)
	GetPartyMarginAccount(market, party, asset string) (*types.Account, error)
	RollbackMarginUpdateOnOrder(ctx context.Context, marketID string, assetID string, transfer *types.Transfer) (*types.LedgerMovement, error)
	GetOrCreatePartyBondAccount(ctx context.Context, partyID, marketID, asset string) (*types.Account, error)
	CreatePartyMarginAccount(ctx context.Context, partyID, marketID, asset string) (string, error)
	FinalSettlement(ctx context.Context, marketID string, transfers []*types.Transfer, factor *num.Uint) ([]*types.LedgerMovement, error)
	ClearMarket(ctx context.Context, mktID, asset string, parties []string, keepInsurance bool) ([]*types.LedgerMovement, error)
	HasGeneralAccount(party, asset string) bool
	ClearPartyMarginAccount(ctx context.Context, party, market, asset string) (*types.LedgerMovement, error)
	CanCoverBond(market, party, asset string, amount *num.Uint) bool
	Hash() []byte
	TransferFeesContinuousTrading(ctx context.Context, marketID string, assetID string, ft events.FeesTransfer) ([]*types.LedgerMovement, error)
	TransferFees(ctx context.Context, marketID string, assetID string, ft events.FeesTransfer) ([]*types.LedgerMovement, error)
	TransferSpotFees(ctx context.Context, marketID string, assetID string, ft events.FeesTransfer) ([]*types.LedgerMovement, error)
	TransferSpotFeesContinuousTrading(ctx context.Context, marketID string, assetID string, ft events.FeesTransfer) ([]*types.LedgerMovement, error)
	MarginUpdate(ctx context.Context, marketID string, updates []events.Risk) ([]*types.LedgerMovement, []events.Margin, []events.Margin, error)
	PerpsFundingSettlement(ctx context.Context, marketID string, transfers []events.Transfer, asset string, round *num.Uint) ([]events.Margin, []*types.LedgerMovement, error)
	MarkToMarket(ctx context.Context, marketID string, transfers []events.Transfer, asset string) ([]events.Margin, []*types.LedgerMovement, error)
	RemoveDistressed(ctx context.Context, parties []events.MarketPosition, marketID, asset string) (*types.LedgerMovement, error)
	GetMarketLiquidityFeeAccount(market, asset string) (*types.Account, error)
	GetAssetQuantum(asset string) (num.Decimal, error)
	GetInsurancePoolBalance(marketID, asset string) (*num.Uint, bool)
	AssetExists(string) bool
	CreateMarketAccounts(context.Context, string, string) (string, string, error)
	CreateSpotMarketAccounts(ctx context.Context, marketID, quoteAsset string) error
	SuccessorInsuranceFraction(ctx context.Context, successor, parent, asset string, fraction num.Decimal) *types.LedgerMovement
	ClearInsurancepool(ctx context.Context, marketID string, asset string, clearFees bool) ([]*types.LedgerMovement, error)
	TransferToHoldingAccount(ctx context.Context, transfer *types.Transfer) (*types.LedgerMovement, error)
	ReleaseFromHoldingAccount(ctx context.Context, transfer *types.Transfer) (*types.LedgerMovement, error)
	ClearSpotMarket(ctx context.Context, mktID, quoteAsset string) ([]*types.LedgerMovement, error)
	PartyHasSufficientBalance(asset, partyID string, amount *num.Uint) error
	TransferSpot(ctx context.Context, partyID, toPartyID, asset string, quantity *num.Uint) (*types.LedgerMovement, error)
	GetOrCreatePartyLiquidityFeeAccount(ctx context.Context, partyID, marketID, asset string) (*types.Account, error)
	GetPartyLiquidityFeeAccount(market, partyID, asset string) (*types.Account, error)
	GetLiquidityFeesBonusDistributionAccount(marketID, asset string) (*types.Account, error)
	CreatePartyGeneralAccount(ctx context.Context, partyID, asset string) (string, error)
	GetOrCreateLiquidityFeesBonusDistributionAccount(ctx context.Context, marketID, asset string) (*types.Account, error)
}

type OrderReferenceCheck types.Order

const (
	// PriceMoveMid used to indicate that the mid price has moved.
	PriceMoveMid = 1

	// PriceMoveBestBid used to indicate that the best bid price has moved.
	PriceMoveBestBid = 2

	// PriceMoveBestAsk used to indicate that the best ask price has moved.
	PriceMoveBestAsk = 4

	// PriceMoveAll used to indicate everything has moved.
	PriceMoveAll = PriceMoveMid + PriceMoveBestBid + PriceMoveBestAsk
)

func (o OrderReferenceCheck) HasMoved(changes uint8) bool {
	return (o.PeggedOrder.Reference == types.PeggedReferenceMid &&
		changes&PriceMoveMid > 0) ||
		(o.PeggedOrder.Reference == types.PeggedReferenceBestBid &&
			changes&PriceMoveBestBid > 0) ||
		(o.PeggedOrder.Reference == types.PeggedReferenceBestAsk &&
			changes&PriceMoveBestAsk > 0)
}

type LiquidityEngine interface {
	ResetSLAEpoch(t time.Time, markPrice *num.Uint, midPrice *num.Uint, positionFactor num.Decimal)
	ApplyPendingProvisions(ctx context.Context, now time.Time) liquidity.Provisions
	PendingProvision() liquidity.Provisions
	PendingProvisionByPartyID(party string) *types.LiquidityProvision
	CalculateSLAPenalties(time.Time) liquidity.SlaPenalties
	ResetAverageLiquidityScores()
	UpdateAverageLiquidityScores(num.Decimal, num.Decimal, *num.Uint, *num.Uint)
	GetAverageLiquidityScores() map[string]num.Decimal
	SubmitLiquidityProvision(context.Context, *types.LiquidityProvisionSubmission, string, liquidity.IDGen) (bool, error)
	RejectLiquidityProvision(context.Context, string) error
	AmendLiquidityProvision(ctx context.Context, lpa *types.LiquidityProvisionAmendment, party string, isCancel bool) (bool, error)
	CancelLiquidityProvision(context.Context, string) error
	ValidateLiquidityProvisionAmendment(*types.LiquidityProvisionAmendment) error
	StopLiquidityProvision(context.Context, string) error
	IsLiquidityProvider(string) bool
	ProvisionsPerParty() liquidity.ProvisionsPerParty
	LiquidityProvisionByPartyID(string) *types.LiquidityProvision
	CalculateSuppliedStake() *num.Uint
	CalculateSuppliedStakeWithoutPending() *num.Uint
	UpdatePartyCommitment(string, *num.Uint) (*types.LiquidityProvision, error)
	EndBlock(*num.Uint, *num.Uint, num.Decimal)
	UpdateMarketConfig(liquidity.RiskModel, liquidity.PriceMonitor, *types.LiquiditySLAParams)
	OnNonPerformanceBondPenaltySlopeUpdate(num.Decimal)
	OnNonPerformanceBondPenaltyMaxUpdate(num.Decimal)
	OnMinProbabilityOfTradingLPOrdersUpdate(num.Decimal)
	OnProbabilityOfTradingTauScalingUpdate(num.Decimal)
	OnMaximumLiquidityFeeFactorLevelUpdate(num.Decimal)
	OnStakeToCcyVolumeUpdate(stakeToCcyVolume num.Decimal)
	IsProbabilityOfTradingInitialised() bool

	SetLastFeeDistributionTime(t time.Time)
	GetLastFeeDistributionTime() time.Time

	Namespace() types.SnapshotNamespace
	Keys() []string
	GetState(k string) ([]byte, []types.StateProvider, error)
	LoadState(ctx context.Context, p *types.Payload) ([]types.StateProvider, error)
	Stopped() bool
	StopSnapshots()
}

type MarketLiquidityEngine interface {
	OnEpochStart(context.Context, time.Time, *num.Uint, *num.Uint, *num.Uint, num.Decimal)
	OnEpochEnd(context.Context, time.Time)
	OnTick(context.Context, time.Time)
	EndBlock(*num.Uint, *num.Uint, num.Decimal)
	SubmitLiquidityProvision(context.Context, *types.LiquidityProvisionSubmission, string, string, types.MarketState) error
	AmendLiquidityProvision(context.Context, *types.LiquidityProvisionAmendment, string, string, types.MarketState) error
	CancelLiquidityProvision(context.Context, string) error
	UpdateMarketConfig(liquidity.RiskModel, liquidity.PriceMonitor, *types.LiquiditySLAParams)
	OnEarlyExitPenalty(num.Decimal)
	OnMinLPStakeQuantumMultiple(num.Decimal)
	OnBondPenaltyFactorUpdate(num.Decimal)
	OnNonPerformanceBondPenaltySlopeUpdate(num.Decimal)
	OnNonPerformanceBondPenaltyMaxUpdate(num.Decimal)
	OnMinProbabilityOfTradingLPOrdersUpdate(num.Decimal)
	OnProbabilityOfTradingTauScalingUpdate(num.Decimal)
	OnMaximumLiquidityFeeFactorLevelUpdate(num.Decimal)
	OnStakeToCcyVolumeUpdate(stakeToCcyVolume num.Decimal)
	StopAllLiquidityProvision(context.Context)
	IsProbabilityOfTradingInitialised() bool
	GetAverageLiquidityScores() map[string]num.Decimal
	ProvisionsPerParty() liquidity.ProvisionsPerParty
	OnMarketClosed(context.Context, time.Time)
	CalculateSuppliedStake() *num.Uint
}

type EquityLikeShares interface {
	AllShares() map[string]num.Decimal
	SetPartyStake(id string, newStakeU *num.Uint)
}

type CommonMarket interface {
	GetID() string
	Hash() []byte
	Reject(context.Context) error
	GetMarketData() types.MarketData
	StartOpeningAuction(context.Context) error
	GetEquityShares() *EquityShares
	IntoType() types.Market
	OnEpochEvent(ctx context.Context, epoch types.Epoch)
	OnEpochRestore(ctx context.Context, epoch types.Epoch)
	GetAssetForProposerBonus() string
	GetMarketCounters() *types.MarketCounters
	GetMarketState() types.MarketState
	BlockEnd(context.Context)
	UpdateMarketState(ctx context.Context, changes *types.MarketStateUpdateConfiguration) error

	// network param updates
	OnMarketPartiesMaximumStopOrdersUpdate(context.Context, *num.Uint)
	OnMarketMinLpStakeQuantumMultipleUpdate(context.Context, num.Decimal)
	OnMarketMinProbabilityOfTradingLPOrdersUpdate(context.Context, num.Decimal)
	OnMarketProbabilityOfTradingTauScalingUpdate(context.Context, num.Decimal)
	OnMarketValueWindowLengthUpdate(time.Duration)
	OnFeeFactorsInfrastructureFeeUpdate(context.Context, num.Decimal)
	OnFeeFactorsMakerFeeUpdate(context.Context, num.Decimal)
	OnMarkPriceUpdateMaximumFrequency(context.Context, time.Duration)
	OnMarketAuctionMinimumDurationUpdate(context.Context, time.Duration)
	OnMarketLiquidityV2EarlyExitPenaltyUpdate(num.Decimal)
	OnMarketLiquidityV2MaximumLiquidityFeeFactorLevelUpdate(num.Decimal)
	OnMarketLiquidityV2SLANonPerformanceBondPenaltySlopeUpdate(num.Decimal)
	OnMarketLiquidityV2SLANonPerformanceBondPenaltyMaxUpdate(num.Decimal)
	OnMarketLiquidityV2StakeToCCYVolume(d num.Decimal)

	// liquidity provision
	CancelLiquidityProvision(context.Context, *types.LiquidityProvisionCancellation, string) error
	AmendLiquidityProvision(context.Context, *types.LiquidityProvisionAmendment, string, string) error
	SubmitLiquidityProvision(context.Context, *types.LiquidityProvisionSubmission, string, string) error

	// order management
	SubmitOrderWithIDGeneratorAndOrderID(context.Context, *types.OrderSubmission, string, IDGenerator, string, bool) (*types.OrderConfirmation, error)
	AmendOrderWithIDGenerator(context.Context, *types.OrderAmendment, string, IDGenerator) (*types.OrderConfirmation, error)
	CancelAllOrders(context.Context, string) ([]*types.OrderCancellationConfirmation, error)
	CancelOrderWithIDGenerator(context.Context, string, string, IDGenerator) (*types.OrderCancellationConfirmation, error)
	CancelAllStopOrders(context.Context, string) error
	CancelStopOrder(context.Context, string, string) error
	SubmitStopOrdersWithIDGeneratorAndOrderIDs(context.Context, *types.StopOrdersSubmission, string, IDGenerator, *string, *string) (*types.OrderConfirmation, error)

	PostRestore(context.Context) error
}

type AccountBalanceChecker interface {
	GetAvailableBalance(party string) (*num.Uint, error)
}

type Teams interface {
	GetTeamMembers(team string, minEpochsInTeam uint64) []string
	GetAllPartiesInTeams(minEpochsInTeam uint64) []string
}
