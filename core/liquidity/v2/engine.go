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
	"encoding/binary"
	"errors"
	"math/rand"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/liquidity/supplied"
	"code.vegaprotocol.io/vega/core/risk"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/types/statevar"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"

	abcitypes "github.com/tendermint/tendermint/abci/types"
)

var (
	ErrLiquidityProvisionDoesNotExist  = errors.New("liquidity provision does not exist")
	ErrLiquidityProvisionAlreadyExists = errors.New("liquidity provision already exists")
	ErrCommitmentAmountIsZero          = errors.New("commitment amount is zero")
	ErrEmptyShape                      = errors.New("liquidity provision contains an empty shape")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/orderbook_mock.go -package mocks code.vegaprotocol.io/vega/core/liquidity OrderBook
type OrderBook interface {
	GetOrdersPerParty(party string) []*types.Order
	GetBestStaticBidPrice() (*num.Uint, error)
	GetBestStaticAskPrice() (*num.Uint, error)
	GetIndicativePrice() *num.Uint
	GetLastTradedPrice() *num.Uint
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/core/liquidity RiskModel,PriceMonitor,IDGen

// Broker - event bus (no mocks needed).
type Broker interface {
	Send(e events.Event)
	SendBatch(evts []events.Event)
}

// TimeService provide the time of the vega node using the tm time.
//
//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/core/liquidity TimeService
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

type slaPerformance struct {
	s                  time.Duration
	start              time.Time
	allEpochsPenalties []num.Decimal
}

type SlaPenalty struct {
	fee, bond num.Decimal
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

func (e *Engine) getMidPrice() (*num.Uint, error) {
	bestBid, err := e.orderBook.GetBestStaticBidPrice()
	if err != nil {
		return nil, err
	}

	bestAsk, err := e.orderBook.GetBestStaticAskPrice()
	if err != nil {
		return nil, err
	}

	two := num.NewUint(2)
	midPrice := num.UintZero()
	if !bestBid.IsZero() && !bestAsk.IsZero() {
		midPrice = midPrice.Div(num.Sum(bestBid, bestAsk), two)
	}

	return midPrice, nil
}

func (e *Engine) doesLPMeetsCommitment(party string) bool {
	lp, ok := e.provisions.Get(party)
	if !ok {
		return false
	}

	one := num.DecimalOne()

	var minPrice, maxPrice num.Decimal
	if e.auctionState.InAuction() {
		priceFactor := num.Min(e.orderBook.GetLastTradedPrice(), e.orderBook.GetIndicativePrice()).ToDecimal()

		// (1.0-market.liquidity.priceRange) x min(last trade price, indicative uncrossing price)
		minPrice = one.Sub(e.priceRange).Mul(priceFactor)
		// (1.0+market.liquidity.priceRange) x max(last trade price, indicative uncrossing price)
		maxPrice = one.Sub(e.priceRange).Mul(priceFactor)
	} else {
		mid, err := e.getMidPrice()
		// if there is no mid price then LP is not meeting their committed volume of notional.
		if err != nil || mid.IsZero() {
			return false
		}

		midD := mid.ToDecimal()
		// (1.0 - market.liquidity.priceRange) x mid
		minPrice = one.Sub(e.priceRange).Mul(midD)
		// (1.0 + market.liquidity.priceRange) x mid
		maxPrice = one.Add(e.priceRange).Mul(midD)
	}

	notionalVolume := num.DecimalZero()
	orders := e.getAllActiveOrders(party)

	for _, o := range orders {
		price := o.Price.ToDecimal()

		// this order is in range and does contribute to the volume on notional
		if price.GreaterThanOrEqual(minPrice) && price.LessThanOrEqual(maxPrice) {
			notionalVolume = notionalVolume.Add(price)
		}
	}

	requiredLiquidity := e.stakeToCcyVolume.Mul(lp.CommitmentAmount.ToDecimal())
	return notionalVolume.GreaterThanOrEqual(requiredLiquidity)
}

func (e *Engine) ResetSLAEpoch(now time.Time) {
	for party, commitment := range e.slaPerformance {
		if e.doesLPMeetsCommitment(party) {
			commitment.start = now
		}

		commitment.s = 0
	}

	e.slaEpochStart = now
}

func (e *Engine) calculateFeePenalty(timeBookFraction num.Decimal) num.Decimal {
	one := num.DecimalOne()

	// TODO karel make this prettier
	return one.Sub(
		timeBookFraction.Sub(e.commitmentMinTimeFraction).Div(one.Sub(e.commitmentMinTimeFraction)),
	).Mul(e.slaCompetitionFactor)
}

func (e *Engine) calculateBondPenalty(timeBookFraction num.Decimal) num.Decimal {
	// TODO karel make this prettier
	min := num.MinD(
		e.nonPerformanceBondPenaltyMax,
		e.nonPerformanceBondPenaltySlope.Mul(num.DecimalOne().Sub(timeBookFraction.Div(e.commitmentMinTimeFraction))),
	)

	return num.MaxD(num.DecimalZero(), min)
}

func (e *Engine) selectFeePenalty(currentPenalty num.Decimal, allPenalties []num.Decimal) num.Decimal {
	l := len(allPenalties)
	if l < 2 {
		return currentPenalty
	}

	performanceHysteresisPeriod := e.performanceHysteresisEpochs - 1
	// Select window windowStart for hysteresis period
	windowStart := l - int(performanceHysteresisPeriod)
	if windowStart < 0 {
		windowStart = 0
	}

	periodAveragePenalty := num.DecimalZero()
	for _, p := range allPenalties[windowStart:] {
		periodAveragePenalty = periodAveragePenalty.Add(p)
	}

	periodAveragePenalty = periodAveragePenalty.Div(num.NewDecimalFromFloat(float64(performanceHysteresisPeriod)))
	return num.MaxD(currentPenalty, periodAveragePenalty)
}

func (e *Engine) CalculateSLAPenalties(now time.Time) {
	observedEpochLength := e.slaEpochStart.Sub(now)

	penaltiesPerParty := map[string]*SlaPenalty{}

	for party, commitment := range e.slaPerformance {
		timeBookFraction := num.DecimalFromInt64(int64(commitment.s / observedEpochLength))

		var feePenalty, bondPenalty num.Decimal

		if timeBookFraction.LessThan(e.commitmentMinTimeFraction) {
			feePenalty = num.DecimalOne()
			bondPenalty = num.DecimalOne()
		} else {
			feePenalty = e.calculateFeePenalty(timeBookFraction)
			bondPenalty = e.calculateBondPenalty(timeBookFraction)
		}

		commitment.allEpochsPenalties = append(commitment.allEpochsPenalties, feePenalty)

		penaltiesPerParty[party] = &SlaPenalty{
			bond: bondPenalty,
			fee:  e.selectFeePenalty(feePenalty, commitment.allEpochsPenalties),
		}
	}

	e.slaPenalties = penaltiesPerParty
}

func (e *Engine) GetSLAPenalties() map[string]*SlaPenalty {
	return e.slaPenalties
}

type TX struct {
	ID string
}

func (t TX) Hash() []byte {
	return crypto.Hash([]byte(t.ID))
}

func (e *Engine) generateKSla(txs []TX) int {
	bytes := []byte{}
	for _, tx := range txs {
		bytes = append(bytes, tx.Hash()...)
	}

	hash := crypto.Hash(bytes)
	seed := binary.BigEndian.Uint64(hash)

	rand.Seed(int64(seed))

	min := 1
	max := len(txs)
	return rand.Intn(max-min+1) + min
}

func (e *Engine) BeginBlock(req abcitypes.RequestBeginBlock, txs []TX) {
	e.kSla = e.generateKSla(txs)
}

func (e *Engine) TxProcessed(txCount int) {
	// Check if the k transaction has been processed
	if e.kSla != txCount {
		return
	}

	for party, commitment := range e.slaPerformance {
		meetsCommitment := e.doesLPMeetsCommitment(party)

		// if LP started meeting commitment
		// else if LP stopped meeting commitment
		if meetsCommitment && commitment.start.IsZero() {
			commitment.start = e.timeService.GetTimeNow()
		} else if !meetsCommitment && !commitment.start.IsZero() {
			commitment.s += e.timeService.GetTimeNow().Sub(commitment.start)
			commitment.start = time.Time{}
		}
	}
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

func (e *Engine) ValidateLiquidityProvisionAmendment(lp *types.LiquidityProvisionAmendment) (err error) {
	if lp.Fee.IsZero() && !lp.ContainsOrders() && (lp.CommitmentAmount == nil || lp.CommitmentAmount.IsZero()) {
		return errors.New("empty liquidity provision amendment content")
	}

	// If orders fee is provided, we need it to be valid
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

	if lp := e.LiquidityProvisionByPartyID(party); lp != nil {
		return ErrLiquidityProvisionAlreadyExists
	}

	var (
		now = e.timeService.GetTimeNow().UnixNano()
		lp  = &types.LiquidityProvision{
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
	)

	// regardless of the final operation (create,update or delete) we finish
	// sending an event.
	defer func() {
		e.broker.Send(events.NewLiquidityProvisionEvent(ctx, lp))
	}()

	e.provisions.Set(party, lp)
	e.slaPerformance[party] = &slaPerformance{}

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

func (e *Engine) OnPriceRange(priceRange num.Decimal) {
	e.priceRange = priceRange
}

func (e *Engine) OnCommitmentMinTimeFraction(commitmentMinTimeFraction num.Decimal) {
	e.commitmentMinTimeFraction = commitmentMinTimeFraction
}

func (e *Engine) OnSlaCompetitionFactor(slaCompetitionFactor num.Decimal) {
	e.slaCompetitionFactor = slaCompetitionFactor
}

func (e *Engine) OnNonPerformanceBondPenaltySlope(nonPerformanceBondPenaltySlope num.Decimal) {
	e.nonPerformanceBondPenaltySlope = nonPerformanceBondPenaltySlope
}
func (e *Engine) OnNonPerformanceBondPenaltyMax(nonPerformanceBondPenaltyMax num.Decimal) {
	e.nonPerformanceBondPenaltyMax = nonPerformanceBondPenaltyMax
}
