package risk

import (
	"context"
	"errors"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/events"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/vegatime"
)

var (
	ErrInsufficientFundsForInitialMargin = errors.New("insufficient funds for initial margin")
	ErrRiskFactorsNotAvailableForAsset   = errors.New("risk factors not available for the specified asset")
)

// Orderbook represent an abstraction over the orderbook
//go:generate go run github.com/golang/mock/mockgen -destination mocks/orderbook_mock.go -package mocks code.vegaprotocol.io/vega/risk Orderbook
type Orderbook interface {
	GetCloseoutPrice(volume uint64, side types.Side) (uint64, error)
}

// AuctionState represents the current auction state of the market, previously we got this information from the matching engine, but really... that's not its job
//go:generate go run github.com/golang/mock/mockgen -destination mocks/auction_state_mock.go -package mocks code.vegaprotocol.io/vega/risk AuctionState
type AuctionState interface {
	InAuction() bool
	AuctionEnd() bool
}

// Broker the event bus broker
//go:generate go run github.com/golang/mock/mockgen -destination mocks/broker_mock.go -package mocks code.vegaprotocol.io/vega/execution Broker
type Broker interface {
	Send(events.Event)
	SendBatch([]events.Event)
}

// AuctionPosition is the enriched market position event (well, it's the same type)
// which is only passed in when the market is in auction mode
type AuctionPosition interface {
	events.MarketPosition
	VWBuy() uint64
	VWSell() uint64
}

type marginChange struct {
	events.Margin // previous event that caused this change
	transfer      *types.Transfer
	margins       *types.MarginLevels
}

// Engine is the risk engine
type Engine struct {
	Config
	marginCalculator *types.MarginCalculator
	log              *logging.Logger
	cfgMu            sync.Mutex
	model            Model
	factors          *types.RiskResult
	waiting          bool
	ob               Orderbook
	as               AuctionState
	broker           Broker

	currTime int64
	mktID    string
}

// NewEngine instantiate a new risk engine
func NewEngine(
	log *logging.Logger,
	config Config,
	marginCalculator *types.MarginCalculator,
	model Model,
	initialFactors *types.RiskResult,
	ob Orderbook,
	as AuctionState,
	broker Broker,
	initialTime int64,
	mktID string,
) *Engine {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	return &Engine{
		log:              log,
		Config:           config,
		marginCalculator: marginCalculator,
		factors:          initialFactors,
		model:            model,
		waiting:          false,
		ob:               ob,
		as:               as,
		broker:           broker,
		currTime:         initialTime,
		mktID:            mktID,
	}
}

func (e *Engine) OnMarginScalingFactorsUpdate(sf *types.ScalingFactors) error {
	if sf.CollateralRelease < sf.InitialMargin || sf.InitialMargin < sf.SearchLevel {
		return errors.New("incompatible margins scaling factors")
	}

	e.marginCalculator.ScalingFactors = sf
	return nil
}

func (e *Engine) OnTimeUpdate(t time.Time) {
	e.currTime = t.UnixNano()
}

// ReloadConf update the internal configuration of the risk engine
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

// CalculateFactors trigger the calculation of the risk factors
func (e *Engine) CalculateFactors(ctx context.Context, now time.Time) {
	// don't calculate risk factors if we are before or at the next update time (calcs are before
	// processing and we calc factors after the time so we wait for time > nextUpdateTime) OR if we are
	// already waiting on risk calcs
	// NB: for continuous risk calcs nextUpdateTime will be 0 so we will always find time > nextUpdateTime
	if e.waiting || now.Before(vegatime.UnixNano(e.factors.NextUpdateTimestamp)) {
		return
	}

	wasCalculated, result := e.model.CalculateRiskFactors(e.factors)
	if wasCalculated {
		e.waiting = false
		if e.model.CalculationInterval() > 0 {
			result.NextUpdateTimestamp = now.Add(e.model.CalculationInterval()).UnixNano()
		}
		e.factors = result
		// FIXME(jeremy): here we are iterating over the risk factors map
		// although we know there's only one asset in the map, we should probably
		// refactor the returned values from the model.
		var rf types.RiskFactor
		for _, v := range result.RiskFactors {
			rf = *v
		}
		rf.Market = e.mktID
		// then we can send in the broker
		e.broker.Send(events.NewRiskFactorEvent(ctx, rf))
	} else {
		e.waiting = true
	}
}

// GetRiskFactors returns risk factors per specified asset if available and an error otherwise
func (e *Engine) GetRiskFactors(asset string) (*types.RiskFactor, error) {
	rf, ok := e.factors.RiskFactors[asset]
	if !ok {
		return nil, ErrRiskFactorsNotAvailableForAsset
	}
	return rf, nil
}

func (e *Engine) UpdateMarginAuction(ctx context.Context, evts []events.Margin, price uint64) ([]events.Risk, []events.Margin, error) {
	if len(evts) == 0 {
		return nil, nil, nil
	}
	revts := make([]events.Risk, 0, len(evts))
	// traders with insufficient margin to meet required level, return the event passed as arg
	low := []events.Margin{}
	eventBatch := make([]events.Event, 0, len(evts))
	// for now, we can assume a single asset for all events
	asset := evts[0].Asset()
	rFactors := *e.factors.RiskFactors[asset]
	for _, evt := range evts {
		levels := e.calculateAuctionMargins(evt, int64(price), rFactors)
		if levels == nil {
			continue
		}
		levels.PartyId = evt.Party()
		levels.Asset = asset // This is assuming there's a single asset at play here
		levels.Timestamp = e.currTime
		levels.MarketId = e.mktID

		curMargin := evt.MarginBalance()
		if curMargin+evt.GeneralBalance() < levels.MaintenanceMargin {
			low = append(low, evt)
			continue
		}
		eventBatch = append(eventBatch, events.NewMarginLevelsEvent(ctx, *levels))
		// trader has sufficient margin, no need to transfer funds
		if curMargin >= levels.InitialMargin {
			continue
		}
		var minAmount uint64
		if levels.MaintenanceMargin > curMargin {
			minAmount = maxUint(levels.MaintenanceMargin-curMargin, 0)
		}
		t := &types.Transfer{
			Owner: evt.Party(),
			Type:  types.TransferType_TRANSFER_TYPE_MARGIN_LOW,
			Amount: &types.FinancialAmount{
				Asset:  asset,
				Amount: levels.InitialMargin - curMargin, // we know curBalance is less than initial
			},
			MinAmount: minAmount,
		}
		revts = append(revts, &marginChange{
			Margin:   evt,
			transfer: t,
			margins:  levels,
		})
	}
	e.broker.SendBatch(eventBatch)
	return revts, low, nil
}

// UpdateMarginOnNewOrder calculate the new margin requirement for a single order
// this is intended to be used when a new order is created in order to ensure the
// trader margin account is at least at the InitialMargin level before the order is added to the book.
func (e *Engine) UpdateMarginOnNewOrder(ctx context.Context, evt events.Margin, markPrice uint64) (events.Risk, error) {
	if evt == nil {
		return nil, nil
	}

	var margins *types.MarginLevels
	if !e.as.InAuction() || e.as.AuctionEnd() {
		margins = e.calculateMargins(evt, int64(markPrice), *e.factors.RiskFactors[evt.Asset()], true, false)
	} else {
		margins = e.calculateAuctionMargins(evt, int64(markPrice), *e.factors.RiskFactors[evt.Asset()])
	}
	// no margins updates, nothing to do then
	if margins == nil {
		return nil, nil
	}

	// update other fields for the margins
	margins.PartyId = evt.Party()
	margins.Asset = evt.Asset()
	margins.Timestamp = e.currTime
	margins.MarketId = e.mktID

	curBalance := evt.MarginBalance()

	// there's not enought monies in the accounts of the party,
	// we break from here. The minimum requires is MAINTENANCE, not INITIAL here!
	if curBalance+evt.GeneralBalance() < margins.MaintenanceMargin {
		return nil, ErrInsufficientFundsForInitialMargin
	}

	// propagate margins levels to the buffer
	e.broker.Send(events.NewMarginLevelsEvent(ctx, *margins))

	// margins are sufficient, nothing to update
	if curBalance >= margins.InitialMargin {
		return nil, nil
	}

	var minAmount uint64
	if margins.MaintenanceMargin > curBalance {
		minAmount = maxUint(margins.MaintenanceMargin-curBalance, 0)
	}

	// margin is < that InitialMargin so we create a transfer request to top it up.
	trnsfr := &types.Transfer{
		Owner: evt.Party(),
		Type:  types.TransferType_TRANSFER_TYPE_MARGIN_LOW,
		Amount: &types.FinancialAmount{
			Asset:  evt.Asset(),
			Amount: margins.InitialMargin - curBalance,
		},
		MinAmount: minAmount, // minimal amount == maintenance
	}

	return &marginChange{
		Margin:   evt,
		transfer: trnsfr,
		margins:  margins,
	}, nil
}

// UpdateMarginsOnSettlement ensure the margin requirement over all positions.
// margins updates are based on the following requirement
//  ---------------------------------------------------------------------------------------
// | 1 | SearchLevel < CurMargin < InitialMargin | nothing to do / no risk for the network |
// | 2 | CurMargin < SearchLevel                 | set margin to InitalLevel               |
// | 3 | CurMargin > ReleaseLevel                | release up to the InitialLevel          |
//  ---------------------------------------------------------------------------------------
// In the case where the CurMargin is smaller to the MaintenanceLevel after trying to
// move monies later, we'll need to close out the trader but that cannot be figured out
// now only in later when we try to move monies from the general account.
func (e *Engine) UpdateMarginsOnSettlement(
	ctx context.Context, evts []events.Margin, markPrice uint64) []events.Risk {
	ret := make([]events.Risk, 0, len(evts))
	// var err error
	// this will keep going until we've closed this channel
	// this can be the result of an error, or being "finished"
	for _, evt := range evts {
		// channel is closed, and we've got a nil interface
		var margins *types.MarginLevels
		if !e.as.InAuction() || e.as.AuctionEnd() {
			margins = e.calculateMargins(evt, int64(markPrice), *e.factors.RiskFactors[evt.Asset()], true, false)
		} else {
			margins = e.calculateAuctionMargins(evt, int64(markPrice), *e.factors.RiskFactors[evt.Asset()])
		}
		// no margins updates, nothing to do then
		if margins == nil {
			continue
		}

		// update other fields for the margins
		margins.Timestamp = e.currTime
		margins.MarketId = e.mktID
		margins.PartyId = evt.Party()
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
		if curMargin >= margins.SearchLevel && curMargin < margins.CollateralReleaseLevel {
			// propagate margins then continue
			e.broker.Send(events.NewMarginLevelsEvent(ctx, *margins))
			continue
		}

		var trnsfr *types.Transfer
		// case 2 -> not enough margin
		if curMargin < margins.SearchLevel {
			var minAmount uint64

			// first calculate minimal amount, which will be specified in the case we are under
			// the maintenance level
			if curMargin < margins.MaintenanceMargin {
				minAmount = margins.MaintenanceMargin - curMargin
			}

			// then the rest is common if we are before or after MaintenanceLevel,
			// we try to reach the InitialMargin level
			trnsfr = &types.Transfer{
				Owner: evt.Party(),
				Type:  types.TransferType_TRANSFER_TYPE_MARGIN_LOW,
				Amount: &types.FinancialAmount{
					Asset:  evt.Asset(),
					Amount: margins.InitialMargin - curMargin,
				},
				MinAmount: minAmount,
			}

		} else { // case 3 -> release some collateral
			trnsfr = &types.Transfer{
				Owner: evt.Party(),
				Type:  types.TransferType_TRANSFER_TYPE_MARGIN_HIGH,
				Amount: &types.FinancialAmount{
					Asset:  evt.Asset(),
					Amount: curMargin - margins.InitialMargin,
				},
				MinAmount: 0,
			}
		}

		// propage margins to the buffers
		e.broker.Send(events.NewMarginLevelsEvent(ctx, *margins))

		risk := &marginChange{
			Margin:   evt,
			transfer: trnsfr,
			margins:  margins,
		}
		ret = append(ret, risk)
	}
	return ret
}

// ExpectMargins is used in the case some traders are in a distressed positions
// in this situation we will only check if the trader margin is > to the maintenance margin
func (e *Engine) ExpectMargins(
	evts []events.Margin, markPrice uint64,
) (okMargins []events.Margin, distressedPositions []events.Margin) {
	okMargins = make([]events.Margin, 0, len(evts)/2)
	distressedPositions = make([]events.Margin, 0, len(evts)/2)
	for _, evt := range evts {
		var margins *types.MarginLevels
		if !e.as.InAuction() || e.as.AuctionEnd() {
			margins = e.calculateMargins(evt, int64(markPrice), *e.factors.RiskFactors[evt.Asset()], false, false)
		} else {
			margins = e.calculateAuctionMargins(evt, int64(markPrice), *e.factors.RiskFactors[evt.Asset()])
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
		if curMargin > margins.MaintenanceMargin {
			okMargins = append(okMargins, evt)
		} else {
			distressedPositions = append(distressedPositions, evt)
		}
	}

	return
}

func (m marginChange) Amount() uint64 {
	if m.transfer == nil {
		return 0
	}
	return m.transfer.Amount.Amount
}

// Transfer - it's actually part of the embedded interface already, but we have to mask it, because this type contains another transfer
func (m marginChange) Transfer() *types.Transfer {
	return m.transfer
}

func (m marginChange) MarginLevels() *types.MarginLevels {
	return m.margins
}

func maxUint(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}
