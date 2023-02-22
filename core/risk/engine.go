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

package risk

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/types/statevar"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	"golang.org/x/exp/maps"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/core/risk Orderbook,AuctionState,TimeService,StateVarEngine,Model

var (
	ErrInsufficientFundsForInitialMargin = errors.New("insufficient funds for initial margin")
	ErrRiskFactorsNotAvailableForAsset   = errors.New("risk factors not available for the specified asset")
)

const RiskFactorStateVarName = "risk-factors"

// Orderbook represent an abstraction over the orderbook.
type Orderbook interface {
	GetCloseoutPrice(volume uint64, side types.Side) (*num.Uint, error)
	GetIndicativePrice() *num.Uint
}

// AuctionState represents the current auction state of the market, previously we got this information from the matching engine, but really... that's not its job.
type AuctionState interface {
	InAuction() bool
	CanLeave() bool
}

// TimeService.
type TimeService interface {
	GetTimeNow() time.Time
}

// Broker the event bus broker.
type Broker interface {
	Send(events.Event)
	SendBatch([]events.Event)
}

type StateVarEngine interface {
	RegisterStateVariable(asset, market, name string, converter statevar.Converter, startCalculation func(string, statevar.FinaliseCalculation), trigger []statevar.EventType, result func(context.Context, statevar.StateVariableResult) error) error
	NewEvent(asset, market string, eventType statevar.EventType)
}

type marginChange struct {
	events.Margin // previous event that caused this change
	transfer      *types.Transfer
	margins       *types.MarginLevels
}

// Engine is the risk engine.
type Engine struct {
	Config
	marginCalculator        *types.MarginCalculator
	scalingFactorsUint      *scalingFactorsUint
	log                     *logging.Logger
	cfgMu                   sync.Mutex
	model                   Model
	factors                 *types.RiskFactor
	waiting                 bool
	ob                      Orderbook
	as                      AuctionState
	timeSvc                 TimeService
	broker                  Broker
	riskFactorsInitialised  bool
	mktID                   string
	asset                   string
	positionFactor          num.Decimal
	linearSlippageFactor    num.Decimal
	quadraticSlippageFactor num.Decimal

	// a map of margin levels events to be send
	// should be flushed after the processing of every transaction
	// partyId -> MarginLevelsEvent
	marginLevelsUpdates map[string]*events.MarginLevels
	updateMarginLevels  func(...*events.MarginLevels)
}

// NewEngine instantiate a new risk engine.
func NewEngine(log *logging.Logger,
	config Config,
	marginCalculator *types.MarginCalculator,
	model Model,
	ob Orderbook,
	as AuctionState,
	timeSvc TimeService,
	broker Broker,
	mktID string,
	asset string,
	stateVarEngine StateVarEngine,
	positionFactor num.Decimal,
	riskFactorsInitialised bool,
	initialisedRiskFactors *types.RiskFactor, // if restored from snapshot, will be nil otherwise
	linearSlippageFactor num.Decimal,
	quadraticSlippageFactor num.Decimal,
) *Engine {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	sfUint := scalingFactorsUintFromDecimals(marginCalculator.ScalingFactors)
	e := &Engine{
		log:                     log,
		Config:                  config,
		marginCalculator:        marginCalculator,
		model:                   model,
		waiting:                 false,
		ob:                      ob,
		as:                      as,
		timeSvc:                 timeSvc,
		broker:                  broker,
		mktID:                   mktID,
		asset:                   asset,
		scalingFactorsUint:      sfUint,
		factors:                 model.DefaultRiskFactors(),
		riskFactorsInitialised:  riskFactorsInitialised,
		positionFactor:          positionFactor,
		linearSlippageFactor:    linearSlippageFactor,
		quadraticSlippageFactor: quadraticSlippageFactor,
		marginLevelsUpdates:     map[string]*events.MarginLevels{},
	}
	stateVarEngine.RegisterStateVariable(asset, mktID, RiskFactorStateVarName, FactorConverter{}, e.startRiskFactorsCalculation, []statevar.EventType{statevar.EventTypeMarketEnactment, statevar.EventTypeMarketUpdated}, e.updateRiskFactor)

	if initialisedRiskFactors != nil {
		e.factors = initialisedRiskFactors
		// we've restored from snapshot, we don't need want to trigger a MarketEnactment event
	} else {
		// trigger the calculation of risk factors for the market
		stateVarEngine.NewEvent(asset, mktID, statevar.EventTypeMarketEnactment)
	}

	e.updateMarginLevels = e.bufferMarginLevels
	if e.StreamMarginLevelsVerbose {
		e.updateMarginLevels = e.sendMarginLevels
	}

	return e
}

func (e *Engine) FlushMarginLevelsEvents() {
	if e.StreamMarginLevelsVerbose || len(e.marginLevelsUpdates) <= 0 {
		return
	}

	e.sendBufferedMarginLevels()
}

func (e *Engine) sendBufferedMarginLevels() {
	parties := maps.Keys(e.marginLevelsUpdates)
	sort.Strings(parties)
	evts := make([]events.Event, 0, len(parties))

	for _, v := range parties {
		evts = append(evts, e.marginLevelsUpdates[v])
	}

	e.broker.SendBatch(evts)
	e.marginLevelsUpdates = make(map[string]*events.MarginLevels, len(e.marginLevelsUpdates))
}

func (e *Engine) sendMarginLevels(m ...*events.MarginLevels) {
	evts := make([]events.Event, 0, len(m))
	for _, ml := range m {
		evts = append(evts, ml)
	}

	e.broker.SendBatch(evts)
}

func (e *Engine) bufferMarginLevels(mls ...*events.MarginLevels) {
	for _, m := range mls {
		e.marginLevelsUpdates[m.PartyID()] = m
	}
}

func (e *Engine) OnMarginScalingFactorsUpdate(sf *types.ScalingFactors) error {
	if sf.CollateralRelease.LessThan(sf.InitialMargin) || sf.InitialMargin.LessThanOrEqual(sf.SearchLevel) {
		return errors.New("incompatible margins scaling factors")
	}

	e.marginCalculator.ScalingFactors = sf
	e.scalingFactorsUint = scalingFactorsUintFromDecimals(sf)
	return nil
}

func (e *Engine) UpdateModel(stateVarEngine StateVarEngine, calculator *types.MarginCalculator, model Model) {
	e.scalingFactorsUint = scalingFactorsUintFromDecimals(calculator.ScalingFactors)
	e.factors = model.DefaultRiskFactors()
	e.model = model
	stateVarEngine.NewEvent(e.asset, e.mktID, statevar.EventTypeMarketUpdated)
}

// ReloadConf update the internal configuration of the risk engine.
func (e *Engine) ReloadConf(cfg Config) {
	e.log.Info("reloading configuration")
	if e.log.GetLevel() != cfg.Level.Get() {
		e.log.Info("updating log level",
			logging.String("old", e.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		e.log.SetLevel(cfg.Level.Get())
	}

	e.cfgMu.Lock()
	e.Config = cfg
	e.cfgMu.Unlock()
}

// GetRiskFactors returns risk factors per specified asset.
func (e *Engine) GetRiskFactors() *types.RiskFactor {
	return e.factors
}

func (e *Engine) UpdateMarginAuction(ctx context.Context, evts []events.Margin, price *num.Uint) ([]events.Risk, []events.Margin) {
	if len(evts) == 0 {
		return nil, nil
	}
	revts := make([]events.Risk, 0, len(evts))
	// parties with insufficient margin to meet required level, return the event passed as arg
	low := []events.Margin{}
	eventBatch := make([]*events.MarginLevels, 0, len(evts))
	// for now, we can assume a single asset for all events
	rFactors := *e.factors
	for _, evt := range evts {
		levels := e.calculateAuctionMargins(evt, price, rFactors)
		if levels == nil {
			continue
		}

		levels.Party = evt.Party()
		levels.Asset = e.asset // This is assuming there's a single asset at play here
		levels.Timestamp = e.timeSvc.GetTimeNow().UnixNano()
		levels.MarketID = e.mktID

		curMargin := evt.MarginBalance()
		if num.Sum(curMargin, evt.GeneralBalance()).LT(levels.MaintenanceMargin) {
			low = append(low, evt)
			continue
		}
		eventBatch = append(eventBatch, events.NewMarginLevelsEvent(ctx, *levels))
		// party has sufficient margin, no need to transfer funds
		if curMargin.GTE(levels.InitialMargin) {
			continue
		}
		minAmount := num.UintZero()
		if levels.MaintenanceMargin.GT(curMargin) {
			minAmount.Sub(levels.MaintenanceMargin, curMargin)
		}
		amt := num.UintZero().Sub(levels.InitialMargin, curMargin) // we know curBalace is less than initial
		t := &types.Transfer{
			Owner: evt.Party(),
			Type:  types.TransferTypeMarginLow,
			Amount: &types.FinancialAmount{
				Asset:  e.asset,
				Amount: amt,
			},
			MinAmount: minAmount,
		}
		revts = append(revts, &marginChange{
			Margin:   evt,
			transfer: t,
			margins:  levels,
		})
	}
	e.updateMarginLevels(eventBatch...)
	return revts, low
}

// UpdateMarginOnNewOrder calculate the new margin requirement for a single order
// this is intended to be used when a new order is created in order to ensure the
// party margin account is at least at the InitialMargin level before the order is added to the book.
func (e *Engine) UpdateMarginOnNewOrder(ctx context.Context, evt events.Margin, markPrice *num.Uint, ord *types.Order) (events.Risk, events.Margin, error) {
	if evt == nil {
		return nil, nil, nil
	}

	var margins *types.MarginLevels
	if !e.as.InAuction() || e.as.CanLeave() {
		margins = e.calculateMargins(evt, markPrice, *e.factors, true, false)
	} else {
		margins = e.calculateAuctionMargins(evt, markPrice, *e.factors)
	}
	// no margins updates, nothing to do then
	if margins == nil {
		return nil, nil, nil
	}

	// update other fields for the margins
	margins.Party = evt.Party()
	margins.Asset = evt.Asset()
	margins.Timestamp = e.timeSvc.GetTimeNow().UnixNano()
	margins.MarketID = e.mktID

	curMarginBalance := evt.MarginBalance()

	softMargin := orderReducesExposure(ord, evt)
	if num.Sum(curMarginBalance, evt.GeneralBalance()).LT(margins.InitialMargin) && !softMargin {
		// there's not enough monies in the accounts of the party
		// and the order does not reduce party's exposure,
		// we break from here. The minimum requirement is INITIAL.
		return nil, nil, ErrInsufficientFundsForInitialMargin
	}

	// propagate margins levels to the buffer
	e.updateMarginLevels(events.NewMarginLevelsEvent(ctx, *margins))

	// margins are sufficient, nothing to update
	if curMarginBalance.GTE(margins.InitialMargin) {
		return nil, nil, nil
	}

	minAmount := num.UintZero()
	if margins.MaintenanceMargin.GT(curMarginBalance) && !softMargin {
		minAmount.Sub(margins.MaintenanceMargin, curMarginBalance)
	}

	// margin is < that InitialMargin so we create a transfer request to top it up.
	trnsfr := &types.Transfer{
		Owner: evt.Party(),
		Type:  types.TransferTypeMarginLow,
		Amount: &types.FinancialAmount{
			Asset:  evt.Asset(),
			Amount: num.UintZero().Sub(margins.InitialMargin, curMarginBalance),
		},
		MinAmount: minAmount, // minimal amount == maintenance
	}

	change := &marginChange{
		Margin:   evt,
		transfer: trnsfr,
		margins:  margins,
	}
	// we don't have enough in general + margin accounts to cover initial margin level, so we'll be dipping into our bond account
	// we have to return the margin event to signal that
	nonBondFunds := num.Sum(curMarginBalance, evt.GeneralBalance())
	nonBondFunds.Sub(nonBondFunds, evt.BondBalance())
	if nonBondFunds.LT(margins.InitialMargin) {
		return change, evt, nil
	}
	return change, nil, nil
}

// UpdateMarginsOnSettlement ensure the margin requirement over all positions.
// margins updates are based on the following requirement
//
//	---------------------------------------------------------------------------------------
//
// | 1 | SearchLevel < CurMargin < InitialMargin | nothing to do / no risk for the network |
// | 2 | CurMargin < SearchLevel                 | set margin to InitialLevel              |
// | 3 | CurMargin > ReleaseLevel                | release up to the InitialLevel          |
//
//	---------------------------------------------------------------------------------------
//
// In the case where the CurMargin is smaller to the MaintenanceLevel after trying to
// move monies later, we'll need to close out the party but that cannot be figured out
// now only in later when we try to move monies from the general account.
func (e *Engine) UpdateMarginsOnSettlement(
	ctx context.Context, evts []events.Margin, markPrice *num.Uint,
) []events.Risk {
	ret := make([]events.Risk, 0, len(evts))
	now := e.timeSvc.GetTimeNow().UnixNano()

	// var err error
	// this will keep going until we've closed this channel
	// this can be the result of an error, or being "finished"
	for _, evt := range evts {
		// before we do anything, see if the position is 0 now, but the margin balance is still set
		// in which case the only response is to release the margin balance.
		if evt.Size() == 0 && evt.Buy() == 0 && evt.Sell() == 0 && !evt.MarginBalance().IsZero() {
			amt := evt.MarginBalance()
			trnsfr := &types.Transfer{
				Owner: evt.Party(),
				Type:  types.TransferTypeMarginHigh,
				Amount: &types.FinancialAmount{
					Asset:  evt.Asset(),
					Amount: amt,
				},
				MinAmount: amt.Clone(),
			}
			margins := types.MarginLevels{
				MaintenanceMargin:      num.UintZero(),
				SearchLevel:            num.UintZero(),
				InitialMargin:          num.UintZero(),
				CollateralReleaseLevel: num.UintZero(),
				Party:                  evt.Party(),
				MarketID:               evt.MarketID(),
				Asset:                  evt.Asset(),
				Timestamp:              now,
			}
			e.updateMarginLevels(events.NewMarginLevelsEvent(ctx, margins))
			ret = append(ret, &marginChange{
				Margin:   evt,
				transfer: trnsfr,
				margins:  &margins,
			})
			continue
		}
		// channel is closed, and we've got a nil interface
		var margins *types.MarginLevels
		if !e.as.InAuction() || e.as.CanLeave() {
			margins = e.calculateMargins(evt, markPrice, *e.factors, true, false)
		} else {
			margins = e.calculateAuctionMargins(evt, markPrice, *e.factors)
		}
		// no margins updates, nothing to do then
		if margins == nil {
			continue
		}

		// update other fields for the margins
		margins.Timestamp = now
		margins.MarketID = e.mktID
		margins.Party = evt.Party()
		margins.Asset = evt.Asset()

		if e.log.GetLevel() == logging.DebugLevel {
			e.log.Debug("margins calculated on settlement",
				logging.String("party-id", evt.Party()),
				logging.String("market-id", evt.MarketID()),
				logging.Reflect("margins", *margins),
			)
		}

		curMargin := evt.MarginBalance()
		// case 1 -> nothing to do margins are sufficient
		if curMargin.GTE(margins.SearchLevel) && curMargin.LT(margins.CollateralReleaseLevel) {
			// propagate margins then continue
			e.updateMarginLevels(events.NewMarginLevelsEvent(ctx, *margins))
			continue
		}

		var trnsfr *types.Transfer
		minAmount := num.UintZero()
		// case 2 -> not enough margin
		if curMargin.LT(margins.SearchLevel) {
			// first calculate minimal amount, which will be specified in the case we are under
			// the maintenance level
			if curMargin.LT(margins.MaintenanceMargin) {
				minAmount.Sub(margins.MaintenanceMargin, curMargin)
			}

			// then the rest is common if we are before or after MaintenanceLevel,
			// we try to reach the InitialMargin level
			trnsfr = &types.Transfer{
				Owner: evt.Party(),
				Type:  types.TransferTypeMarginLow,
				Amount: &types.FinancialAmount{
					Asset:  evt.Asset(),
					Amount: num.UintZero().Sub(margins.InitialMargin, curMargin),
				},
				MinAmount: minAmount,
			}
		} else { // case 3 -> release some collateral
			// collateral not relased in auction
			if e.as.InAuction() && !e.as.CanLeave() {
				// propagate margins then continue
				e.updateMarginLevels(events.NewMarginLevelsEvent(ctx, *margins))
				continue
			}
			trnsfr = &types.Transfer{
				Owner: evt.Party(),
				Type:  types.TransferTypeMarginHigh,
				Amount: &types.FinancialAmount{
					Asset:  evt.Asset(),
					Amount: num.UintZero().Sub(curMargin, margins.InitialMargin),
				},
				MinAmount: minAmount,
			}
		}

		// propage margins to the buffers
		e.updateMarginLevels(events.NewMarginLevelsEvent(ctx, *margins))

		risk := &marginChange{
			Margin:   evt,
			transfer: trnsfr,
			margins:  margins,
		}
		ret = append(ret, risk)
	}
	return ret
}

// ExpectMargins is used in the case some parties are in a distressed positions
// in this situation we will only check if the party margin is > to the maintenance margin.
func (e *Engine) ExpectMargins(
	evts []events.Margin, markPrice *num.Uint,
) (okMargins []events.Margin, distressedPositions []events.Margin) {
	okMargins = make([]events.Margin, 0, len(evts)/2)
	distressedPositions = make([]events.Margin, 0, len(evts)/2)
	for _, evt := range evts {
		var margins *types.MarginLevels
		if !e.as.InAuction() || e.as.CanLeave() {
			margins = e.calculateMargins(evt, markPrice, *e.factors, false, false)
		} else {
			margins = e.calculateAuctionMargins(evt, markPrice, *e.factors)
		}
		// no margins updates, nothing to do then
		if margins == nil {
			okMargins = append(okMargins, evt)
			continue
		}
		if e.log.GetLevel() == logging.DebugLevel {
			e.log.Debug("margins calculated",
				logging.String("party-id", evt.Party()),
				logging.String("market-id", evt.MarketID()),
				logging.Reflect("margins", *margins),
			)
		}

		curMargin := evt.MarginBalance()
		if curMargin.GT(margins.MaintenanceMargin) {
			okMargins = append(okMargins, evt)
		} else {
			distressedPositions = append(distressedPositions, evt)
		}
	}

	return okMargins, distressedPositions
}

func (m marginChange) Amount() *num.Uint {
	if m.transfer == nil {
		return nil
	}
	return m.transfer.Amount.Amount.Clone()
}

// Transfer - it's actually part of the embedded interface already, but we have to mask it, because this type contains another transfer.
func (m marginChange) Transfer() *types.Transfer {
	return m.transfer
}

func (m marginChange) MarginLevels() *types.MarginLevels {
	return m.margins
}

func orderReducesExposure(ord *types.Order, pos events.Margin) bool {
	if ord == nil || pos.Size() == 0 {
		return false
	}
	// long position and short order
	if pos.Size() > 0 && ord.Side == types.SideSell {
		// market order reduces exposure and doesn't flip position to the other side
		if pos.Size()-int64(ord.Remaining) >= 0 && ord.Type == types.OrderTypeMarket {
			return true
		}
		// sum of all short limit orders wouldn't flip the position if filled (ord already included in pos)
		if pos.Size()-pos.Sell() >= 0 {
			return true
		}
	}
	// short position and long order
	if pos.Size() < 0 && ord.Side == types.SideBuy {
		// market order reduces exposure and doesn't flip position to the other side
		if pos.Size()+int64(ord.Remaining) <= 0 && ord.Type == types.OrderTypeMarket {
			return true
		}
		// sum of all long limit orders wouldn't flip the position if filled (ord already included in pos)
		if pos.Size()+pos.Buy() <= 0 {
			return true
		}
	}
	return false
}
