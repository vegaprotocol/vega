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
	"code.vegaprotocol.io/vega/core/risk"
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
}

type TargetStateFunc func() *num.Uint

// shell we delete it after deleting the LP
type slaPerformance struct {
	s                  time.Duration
	start              time.Time
	allEpochsPenalties []num.Decimal
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
	provisions *SnapshotableProvisionsPerParty

	// this is the max fee that can be specified
	maxFee num.Decimal

	// fields used for liquidity score calculation (quality of deployed orders)
	avgScores map[string]num.Decimal
	nAvg      int64 // counter for the liquidity score running average

	getTargetStake TargetStateFunc

	// sla related net params
	stakeToCcyVolume               num.Decimal
	priceRange                     num.Decimal
	commitmentMinTimeFraction      num.Decimal
	slaCompetitionFactor           num.Decimal
	nonPerformanceBondPenaltySlope num.Decimal
	nonPerformanceBondPenaltyMax   num.Decimal

	performanceHysteresisEpochs uint

	// fields related to SLA commitment
	slaPerformance map[string]*slaPerformance
	slaPenalties   map[string]*SlaPenalty
	kSla           int

	slaEpochStart time.Time
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
	stakeToCcyVolume num.Decimal,
	priceRange num.Decimal,
	commitmentMinTimeFraction num.Decimal,
	slaCompetitionFactor num.Decimal,
	nonPerformanceBondPenaltySlope num.Decimal,
	nonPerformanceBondPenaltyMax num.Decimal,
	performanceHysteresisEpochs uint,
	getTargetStake TargetStateFunc,
) *Engine {
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())
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
		provisions: newSnapshotableProvisionsPerParty(),

		getTargetStake: getTargetStake,

		// SLA commitment
		slaPerformance: map[string]*slaPerformance{},

		stakeToCcyVolume:               stakeToCcyVolume,
		priceRange:                     priceRange,
		commitmentMinTimeFraction:      commitmentMinTimeFraction,
		slaCompetitionFactor:           slaCompetitionFactor,
		nonPerformanceBondPenaltySlope: nonPerformanceBondPenaltySlope,
		nonPerformanceBondPenaltyMax:   nonPerformanceBondPenaltyMax,
		performanceHysteresisEpochs:    performanceHysteresisEpochs,
	}
	e.ResetAverageLiquidityScores() // initialise

	return e
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

// GetInactiveParties returns a set of all the parties
// with inactive commitment.
// @TODO karel change this function to use real orderbook
func (e *Engine) GetInactiveParties() map[string]struct{} {
	ret := map[string]struct{}{}
	for _, p := range e.provisions.ProvisionsPerParty {
		if p.Status != types.LiquidityProvisionStatusActive {
			ret[p.Party] = struct{}{}
		}
	}
	return ret
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

	// now delete all stuff
	e.provisions.Delete(party)
	delete(e.slaPerformance, party)
	return nil
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
	// if fee, err := strconv.ParseFloat(lp.Fee, 64); err != nil || fee < 0 || len(lp.Fee) <= 0 || fee > e.maxFee {
	if lp.Fee.IsNegative() || lp.Fee.GreaterThan(e.maxFee) {
		return errors.New("invalid liquidity provision fee")
	}

	return nil
}

func (e *Engine) rejectLiquidityProvisionSubmission(ctx context.Context, lps *types.LiquidityProvisionSubmission, party, id string) {
	// here we just build a liquidityProvision and set its
	// status to rejected before sending it through the bus
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

// SubmitLiquidityProvision handles a new liquidity provision submission.
// It's used to create, update or delete a LiquidityProvision.
// The LiquidityProvision is created if submitted for the first time, updated if a
// previous one was created for the same PartyId or deleted (if exists) when
// the CommitmentAmount is set to 0.
func (e *Engine) SubmitLiquidityProvision(
	ctx context.Context,
	lps *types.LiquidityProvisionSubmission,
	party string,
	idgen IDGen,
) error {
	if err := e.ValidateLiquidityProvisionSubmission(lps, false); err != nil {
		e.rejectLiquidityProvisionSubmission(ctx, lps, party, idgen.NextID())
		return err
	}

	if foundLp := e.LiquidityProvisionByPartyID(party); foundLp != nil {
		return ErrLiquidityProvisionAlreadyExists
	}

	now := e.timeService.GetTimeNow().UnixNano()
	lp := &types.LiquidityProvision{
		ID:               idgen.NextID(),
		MarketID:         lps.MarketID,
		Party:            party,
		CreatedAt:        now,
		Fee:              lps.Fee,
		Status:           types.LiquidityProvisionStatusActive,
		Reference:        lps.Reference,
		Version:          1,
		CommitmentAmount: lps.CommitmentAmount,
		UpdatedAt:        now,
	}

	e.provisions.Set(party, lp)
	e.slaPerformance[party] = &slaPerformance{}

	e.broker.Send(events.NewLiquidityProvisionEvent(ctx, lp))
	return nil
}

// LiquidityProvisionByPartyID returns the LP associated to a Party if any.
// If not, it returns nil.
func (e *Engine) LiquidityProvisionByPartyID(partyID string) *types.LiquidityProvision {
	lp, _ := e.provisions.Get(partyID)
	return lp
}

// CalculateSuppliedStake returns the sum of commitment amounts from all the liquidity providers.
func (e *Engine) CalculateSuppliedStake() *num.Uint {
	ss := num.UintZero()
	for _, v := range e.provisions.ProvisionsPerParty {
		ss.AddSum(v.CommitmentAmount)
	}
	return ss
}

func (e *Engine) IsPoTInitialised() bool {
	return e.suppliedEngine.IsPoTInitialised()
}

func (e *Engine) UpdateMarketConfig(model risk.Model, monitor PriceMonitor) {
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

func (e *Engine) OnPriceRangeUpdate(priceRange num.Decimal) {
	e.priceRange = priceRange
}

func (e *Engine) OnCommitmentMinTimeFractionUpdate(commitmentMinTimeFraction num.Decimal) {
	e.commitmentMinTimeFraction = commitmentMinTimeFraction
}

func (e *Engine) OnSlaCompetitionFactorUpdate(slaCompetitionFactor num.Decimal) {
	e.slaCompetitionFactor = slaCompetitionFactor
}

func (e *Engine) OnNonPerformanceBondPenaltySlopeUpdate(nonPerformanceBondPenaltySlope num.Decimal) {
	e.nonPerformanceBondPenaltySlope = nonPerformanceBondPenaltySlope
}

func (e *Engine) OnNonPerformanceBondPenaltyMaxUpdate(nonPerformanceBondPenaltyMax num.Decimal) {
	e.nonPerformanceBondPenaltyMax = nonPerformanceBondPenaltyMax
}

func (e *Engine) OnPerformanceHysteresisEpochsUpdate(performanceHysteresisEpochs uint) {
	e.performanceHysteresisEpochs = performanceHysteresisEpochs
}
