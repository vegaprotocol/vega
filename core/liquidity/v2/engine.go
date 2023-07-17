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

package liquidity

import (
	"context"
	"errors"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/liquidity/supplied"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/types/statevar"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

var (
	ErrLiquidityProvisionDoesNotExist  = errors.New("liquidity provision does not exist")
	ErrLiquidityProvisionAlreadyExists = errors.New("liquidity provision already exists")
	ErrCommitmentAmountIsZero          = errors.New("commitment amount is zero")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/orderbook_mock.go -package mocks code.vegaprotocol.io/vega/core/liquidity/v2 OrderBook
type OrderBook interface {
	GetOrdersPerParty(party string) []*types.Order
	GetBestStaticBidPrice() (*num.Uint, error)
	GetBestStaticAskPrice() (*num.Uint, error)
	GetIndicativePrice() *num.Uint
	GetLastTradedPrice() *num.Uint
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/core/liquidity/v2 RiskModel,PriceMonitor,IDGen

// Broker - event bus (no mocks needed).
type Broker interface {
	Send(e events.Event)
	SendBatch(evts []events.Event)
}

// TimeService provide the time of the vega node using the tm time.
//
//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/core/liquidity/v2 TimeService
type TimeService interface {
	GetTimeNow() time.Time
}

// RiskModel allows calculation of min/max price range and a probability of trading.
type RiskModel interface {
	ProbabilityOfTrading(currentPrice, orderPrice num.Decimal, minPrice, maxPrice num.Decimal, yFrac num.Decimal, isBid, applyMinMax bool) num.Decimal
	GetProjectionHorizon() num.Decimal
}

// PriceMonitor provides the range of valid prices, that is prices that
// wouldn't trade the current trading mode.
type PriceMonitor interface {
	GetValidPriceRange() (num.WrappedDecimal, num.WrappedDecimal)
}

// IDGen is an id generator for orders.
type IDGen interface {
	NextID() string
}

type StateVarEngine interface {
	RegisterStateVariable(asset, market, name string, converter statevar.Converter, startCalculation func(string, statevar.FinaliseCalculation), trigger []statevar.EventType, result func(context.Context, statevar.StateVariableResult) error) error
}

type AuctionState interface {
	InAuction() bool
	IsOpeningAuction() bool
}

type slaPerformance struct {
	s                 time.Duration
	start             time.Time
	previousPenalties *sliceRing[*num.Decimal]
}

type SlaPenalties struct {
	AllPartiesHaveFullFeePenalty bool
	PenaltiesPerParty            map[string]*SlaPenalty
}

type SlaPenalty struct {
	Fee, Bond num.Decimal
}

// Engine handles Liquidity provision.
type Engine struct {
	marketID       string
	log            *logging.Logger
	timeService    TimeService
	broker         Broker
	suppliedEngine *supplied.Engine
	orderBook      OrderBook
	auctionState   AuctionState

	// state
	provisions        *SnapshotableProvisionsPerParty
	pendingProvisions *SnapshotablePendingProvisions

	// this is the max fee that can be specified
	maxFee num.Decimal

	// fields used for liquidity score calculation (quality of deployed orders)
	avgScores map[string]num.Decimal
	nAvg      int64 // counter for the liquidity score running average

	// sla related net params
	stakeToCcyVolume               num.Decimal
	nonPerformanceBondPenaltySlope num.Decimal
	nonPerformanceBondPenaltyMax   num.Decimal

	openPlusPriceRange  num.Decimal
	openMinusPriceRange num.Decimal
	slaParams           *types.LiquiditySLAParams

	// fields related to SLA commitment
	slaPerformance map[string]*slaPerformance
	slaEpochStart  time.Time
}

// NewEngine returns a new Liquidity Engine.
func NewEngine(config Config,
	log *logging.Logger,
	timeService TimeService,
	broker Broker,
	riskModel RiskModel,
	priceMonitor PriceMonitor,
	orderBook OrderBook,
	auctionState AuctionState,
	asset string,
	marketID string,
	stateVarEngine StateVarEngine,
	positionFactor num.Decimal,
	slaParams *types.LiquiditySLAParams,
) *Engine {
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	one := num.DecimalOne()

	e := &Engine{
		marketID:    marketID,
		log:         log,
		timeService: timeService,
		broker:      broker,
		// tick size to be used by the supplied engine should actually be in asset decimal
		suppliedEngine: supplied.NewEngine(riskModel, priceMonitor, asset, marketID, stateVarEngine, log, positionFactor),
		orderBook:      orderBook,
		auctionState:   auctionState,

		// parameters
		maxFee: num.DecimalFromInt64(1),

		// provisions related state
		provisions:        newSnapshotableProvisionsPerParty(),
		pendingProvisions: newSnapshotablePendingProvisions(),

		// SLA commitment
		slaPerformance: map[string]*slaPerformance{},

		openPlusPriceRange:  one.Add(slaParams.PriceRange),
		openMinusPriceRange: one.Sub(slaParams.PriceRange),
		slaParams:           slaParams,
	}
	e.ResetAverageLiquidityScores() // initialise

	return e
}

// SubmitLiquidityProvision handles a new liquidity provision submission.
// Returns whether or not submission has been applied immediately.
func (e *Engine) SubmitLiquidityProvision(
	ctx context.Context,
	lps *types.LiquidityProvisionSubmission,
	party string,
	idgen IDGen,
) (bool, error) {
	if err := e.ValidateLiquidityProvisionSubmission(lps, false); err != nil {
		e.rejectLiquidityProvisionSubmission(ctx, lps, party, idgen.NextID())
		return false, err
	}

	if foundLp := e.LiquidityProvisionByPartyID(party); foundLp != nil {
		return false, ErrLiquidityProvisionAlreadyExists
	}

	now := e.timeService.GetTimeNow().UnixNano()
	provision := &types.LiquidityProvision{
		ID:               idgen.NextID(),
		MarketID:         lps.MarketID,
		Party:            party,
		CreatedAt:        now,
		Fee:              lps.Fee,
		Status:           types.LiquidityProvisionStatusPending,
		Reference:        lps.Reference,
		Version:          1,
		CommitmentAmount: lps.CommitmentAmount,
		UpdatedAt:        now,
	}

	e.broker.Send(events.NewLiquidityProvisionEvent(ctx, provision))

	// add immediately during the opening auction
	// otherwise schedule to be added at the beginning of new epoch
	if e.auctionState.IsOpeningAuction() {
		e.provisions.Set(party, provision)
		e.slaPerformance[party] = &slaPerformance{
			previousPenalties: NewSliceRing[*num.Decimal](e.slaParams.PerformanceHysteresisEpochs),
		}
		return true, nil
	}

	provision.Status = types.LiquidityProvisionStatusPending
	e.pendingProvisions.Set(party, provision)
	return false, nil
}

func (e *Engine) ApplyPendingProvisions(ctx context.Context, now time.Time) map[string]*types.LiquidityProvision {
	updatedProvisionsPerParty := make(map[string]*types.LiquidityProvision, e.pendingProvisions.Len())

	for _, party := range e.pendingProvisions.sortedKeys() {
		provision, _ := e.pendingProvisions.Get(party)
		updatedProvisionsPerParty[party] = provision
		provision.UpdatedAt = now.UnixNano()

		// if commitment was reduced to 0, all party provision related data can be deleted
		// otherwise we apply the new commitment
		if provision.CommitmentAmount.IsZero() {
			provision.Status = types.LiquidityProvisionStatusStopped
			e.destroyProvision(party)
		} else {
			provision.Status = types.LiquidityProvisionStatusActive
			e.provisions.Set(party, provision)
			if _, ok := e.slaPerformance[party]; !ok {
				e.slaPerformance[party] = &slaPerformance{
					previousPenalties: NewSliceRing[*num.Decimal](e.slaParams.PerformanceHysteresisEpochs),
				}
			}
		}

		e.broker.Send(events.NewLiquidityProvisionEvent(ctx, provision))
	}

	e.pendingProvisions = newSnapshotablePendingProvisions()
	return updatedProvisionsPerParty
}

func (e *Engine) PendingProvisionByPartyID(party string) *types.LiquidityProvision {
	provision, _ := e.pendingProvisions.Get(party)
	return provision
}

func (e *Engine) PendingProvision() map[string]*types.LiquidityProvision {
	return e.pendingProvisions.PendingProvisions
}

// RejectLiquidityProvision removes a parties commitment of liquidity.
func (e *Engine) RejectLiquidityProvision(ctx context.Context, party string) error {
	return e.stopLiquidityProvision(
		ctx, party, types.LiquidityProvisionStatusRejected)
}

// CancelLiquidityProvision removes a parties commitment of liquidity
// Returns the liquidityOrders if any.
func (e *Engine) CancelLiquidityProvision(ctx context.Context, party string) error {
	return e.stopLiquidityProvision(
		ctx, party, types.LiquidityProvisionStatusCancelled)
}

// StopLiquidityProvision removes a parties commitment of liquidity
// Returns the liquidityOrders if any.
func (e *Engine) StopLiquidityProvision(ctx context.Context, party string) error {
	return e.stopLiquidityProvision(
		ctx, party, types.LiquidityProvisionStatusStopped)
}

func (e *Engine) ValidateLiquidityProvisionSubmission(
	lp *types.LiquidityProvisionSubmission,
	zeroCommitmentIsValid bool,
) (err error) {
	// we check if the commitment is 0 which would mean this is a cancel
	// a cancel does not need validations
	if lp.CommitmentAmount.IsZero() {
		if zeroCommitmentIsValid {
			return nil
		}
		return ErrCommitmentAmountIsZero
	}

	// not sure how to check for a missing fee, 0 could be valid
	// then again, that validation should've happened before reaching this point
	if lp.Fee.IsNegative() || lp.Fee.GreaterThan(e.maxFee) {
		return errors.New("invalid liquidity provision fee")
	}

	return nil
}

func (e *Engine) stopLiquidityProvision(
	ctx context.Context, party string, status types.LiquidityProvisionStatus,
) error {
	lp, ok := e.provisions.Get(party)
	if !ok {
		return errors.New("party have no liquidity provision orders")
	}

	now := e.timeService.GetTimeNow().UnixNano()

	lp.Status = status
	lp.UpdatedAt = now
	e.broker.Send(events.NewLiquidityProvisionEvent(ctx, lp))

	// now delete all party related data stuff
	e.destroyProvision(party)
	return nil
}

func (e *Engine) destroyProvision(party string) {
	e.provisions.Delete(party)
	delete(e.slaPerformance, party)
	e.pendingProvisions.Delete(party)
}

func (e *Engine) rejectLiquidityProvisionSubmission(ctx context.Context, lps *types.LiquidityProvisionSubmission, party, id string) {
	lp := &types.LiquidityProvision{
		ID:               id,
		Fee:              lps.Fee,
		MarketID:         lps.MarketID,
		Party:            party,
		Status:           types.LiquidityProvisionStatusRejected,
		CreatedAt:        e.timeService.GetTimeNow().UnixNano(),
		CommitmentAmount: lps.CommitmentAmount.Clone(),
		Reference:        lps.Reference,
	}

	e.broker.Send(events.NewLiquidityProvisionEvent(ctx, lp))
}

// IsLiquidityProvider returns true if the party hold any liquidity commitment.
func (e *Engine) IsLiquidityProvider(party string) bool {
	_, ok := e.provisions.Get(party)
	return ok
}

// ProvisionsPerParty returns the registered a map of party-id -> LiquidityProvision.
func (e *Engine) ProvisionsPerParty() ProvisionsPerParty {
	return e.provisions.ProvisionsPerParty
}

// LiquidityProvisionByPartyID returns the LP associated to a Party if any.
// If not, it returns nil.
func (e *Engine) LiquidityProvisionByPartyID(partyID string) *types.LiquidityProvision {
	lp, _ := e.provisions.Get(partyID)
	return lp
}

// UpdatePartyCommitment allows to change party commitment.
// It should be used for synchronizing commitment with bond account.
func (e *Engine) UpdatePartyCommitment(partyID string, newCommitment *num.Uint) (*types.LiquidityProvision, error) {
	lp, ok := e.provisions.Get(partyID)
	if !ok {
		return nil, ErrLiquidityProvisionDoesNotExist
	}

	lp.CommitmentAmount = newCommitment.Clone()
	e.provisions.Set(partyID, lp)
	return lp, nil
}

// CalculateSuppliedStake returns the sum of commitment amounts from all the liquidity providers.
// Includes pending commitment if they are greater then the original one.
func (e *Engine) CalculateSuppliedStake() *num.Uint {
	supplied := num.UintZero()

	for party, pending := range e.pendingProvisions.PendingProvisions {
		provision, ok := e.provisions.Get(party)
		if ok && pending.CommitmentAmount.LT(provision.CommitmentAmount) {
			supplied.AddSum(provision.CommitmentAmount)
			continue
		}
		supplied.AddSum(pending.CommitmentAmount)
	}

	for party, provision := range e.provisions.ProvisionsPerParty {
		_, ok := e.pendingProvisions.Get(party)
		if ok {
			continue
		}

		supplied.AddSum(provision.CommitmentAmount)
	}

	return supplied
}

// CalculateSuppliedStakeWithoutPending returns the sum of commitment amounts
// from all the liquidity providers. Does not include pending commitments.
func (e *Engine) CalculateSuppliedStakeWithoutPending() *num.Uint {
	supplied := num.UintZero()
	for _, provision := range e.provisions.ProvisionsPerParty {
		supplied.AddSum(provision.CommitmentAmount)
	}
	return supplied
}

func (e *Engine) IsProbabilityOfTradingInitialised() bool {
	return e.suppliedEngine.IsProbabilityOfTradingInitialised()
}

func (e *Engine) UpdateMarketConfig(model RiskModel, monitor PriceMonitor, slaParams *types.LiquiditySLAParams) {
	e.onSLAParamsUpdate(slaParams)
	e.suppliedEngine.UpdateMarketConfig(model, monitor)
}

func (e *Engine) SetGetStaticPricesFunc(f func() (num.Decimal, num.Decimal, error)) {
	e.suppliedEngine.SetGetStaticPricesFunc(f)
}

func (e *Engine) OnMinProbabilityOfTradingLPOrdersUpdate(v num.Decimal) {
	e.suppliedEngine.OnMinProbabilityOfTradingLPOrdersUpdate(v)
}

func (e *Engine) OnProbabilityOfTradingTauScalingUpdate(v num.Decimal) {
	e.suppliedEngine.OnProbabilityOfTradingTauScalingUpdate(v)
}

func (e *Engine) OnMaximumLiquidityFeeFactorLevelUpdate(f num.Decimal) {
	e.maxFee = f
}

func (e *Engine) OnStakeToCcyVolumeUpdate(stakeToCcyVolume num.Decimal) {
	e.stakeToCcyVolume = stakeToCcyVolume
}

func (e *Engine) OnNonPerformanceBondPenaltySlopeUpdate(nonPerformanceBondPenaltySlope num.Decimal) {
	e.nonPerformanceBondPenaltySlope = nonPerformanceBondPenaltySlope
}

func (e *Engine) OnNonPerformanceBondPenaltyMaxUpdate(nonPerformanceBondPenaltyMax num.Decimal) {
	e.nonPerformanceBondPenaltyMax = nonPerformanceBondPenaltyMax
}

func (e *Engine) onSLAParamsUpdate(slaParams *types.LiquiditySLAParams) {
	one := num.DecimalOne()
	e.openPlusPriceRange = one.Add(slaParams.PriceRange)
	e.openMinusPriceRange = one.Sub(slaParams.PriceRange)
	if e.slaParams.PerformanceHysteresisEpochs != slaParams.PerformanceHysteresisEpochs {
		for _, performance := range e.slaPerformance {
			performance.previousPenalties.ModifySize(e.slaParams.PerformanceHysteresisEpochs)
		}
	}
	e.slaParams = slaParams
}
