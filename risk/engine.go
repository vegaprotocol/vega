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
)

// Orderbook represent an abstraction over the orderbook
//go:generate go run github.com/golang/mock/mockgen -destination mocks/orderbook_mock.go -package mocks code.vegaprotocol.io/vega/risk Orderbook
type Orderbook interface {
	GetCloseoutPrice(volume uint64, side types.Side) (uint64, error)
}

// MarginLevelsBuf ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/margin_levels_buf_mock.go -package mocks code.vegaprotocol.io/vega/execution MarginLevelsBuf
type MarginLevelsBuf interface {
	Add(types.MarginLevels)
}

type marginChange struct {
	events.Margin       // previous event that caused this change
	amount        int64 // the amount we need to move (positive is move to margin, neg == move to general)
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
	marginsLevelsBuf MarginLevelsBuf

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
	mlBuf MarginLevelsBuf,
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
		marginsLevelsBuf: mlBuf,
		currTime:         initialTime,
		mktID:            mktID,
	}
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
func (e *Engine) CalculateFactors(now time.Time) {
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
	} else {
		e.waiting = true
	}
}

// UpdateMarginOnNewOrder calculate the new margin requirement for a single order
// this is intended to be used when a new order is created in order to ensure the
// trader margin account is at least at the InitialMargin level before the order is added to the book.
func (e *Engine) UpdateMarginOnNewOrder(evt events.Margin, markPrice uint64) (events.Risk, error) {
	if evt == nil {
		return nil, nil
	}

	margins := e.calculateMargins(evt, int64(markPrice), *e.factors.RiskFactors[evt.Asset()], true)
	// no margins updates, nothing to do then
	if margins == nil {
		return nil, nil
	}
	if e.log.GetLevel() == logging.DebugLevel {
		e.log.Debug("margins calculated on new order",
			logging.String("party-id", evt.Party()),
			logging.String("market-id", evt.MarketID()),
			logging.Reflect("margins", *margins),
		)
	}

	// update other fields for the margins
	margins.PartyID = evt.Party()
	margins.Asset = evt.Asset()
	margins.Timestamp = e.currTime
	margins.MarketID = e.mktID

	curBalance := evt.MarginBalance()

	// there's not enought monies in the accounts of the party,
	// we break from here
	if int64(evt.MarginBalance()+evt.GeneralBalance()) < margins.InitialMargin {
		return nil, ErrInsufficientFundsForInitialMargin
	}

	// propagate margins levels to the buffer
	e.marginsLevelsBuf.Add(*margins)

	// margins are sufficient, nothing to update
	if int64(curBalance) >= margins.InitialMargin {
		return nil, nil
	}

	// margin is < that InitialMargin so we create a transfer request to top it up.
	trnsfr := &types.Transfer{
		Owner: evt.Party(),
		Size:  1,
		Type:  types.TransferType_MARGIN_LOW,
		Amount: &types.FinancialAmount{
			Asset:     evt.Asset(),
			Amount:    margins.InitialMargin - int64(curBalance),
			MinAmount: margins.InitialMargin - int64(curBalance),
		},
	}

	return &marginChange{
		Margin:   evt,
		amount:   trnsfr.Amount.Amount,
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
		margins := e.calculateMargins(evt, int64(markPrice), *e.factors.RiskFactors[evt.Asset()], true)
		// no margins updates, nothing to do then
		if margins == nil {
			continue
		}

		// update other fields for the margins
		margins.Timestamp = e.currTime
		margins.MarketID = e.mktID
		margins.PartyID = evt.Party()
		margins.Asset = evt.Asset()

		if e.log.GetLevel() == logging.DebugLevel {
			e.log.Debug("margins calculated on settlement",
				logging.String("party-id", evt.Party()),
				logging.String("market-id", evt.MarketID()),
				logging.Reflect("margins", *margins),
			)
		}

		curMargin := int64(evt.MarginBalance())
		// case 1 -> nothing to do margins are sufficient
		if curMargin >= margins.SearchLevel && curMargin < margins.CollateralReleaseLevel {
			// propagate margins then continue
			e.marginsLevelsBuf.Add(*margins)
			continue
		}

		var trnsfr *types.Transfer
		// case 2 -> not enough margin
		if curMargin <= margins.SearchLevel {
			// first calculate minimal amount, which will be specified in the case we are under
			// the maintenance level
			var minAmount int64
			if curMargin < margins.MaintenanceMargin {
				minAmount = margins.SearchLevel - curMargin
			}

			// then the rest is common if we are before or after MaintenanceLevel,
			// we try to reach the InitialMargin level
			trnsfr = &types.Transfer{
				Owner: evt.Party(),
				Size:  1,
				Type:  types.TransferType_MARGIN_LOW,
				Amount: &types.FinancialAmount{
					Asset:     evt.Asset(),
					Amount:    margins.InitialMargin - curMargin,
					MinAmount: minAmount,
				},
			}

		} else if curMargin >= margins.CollateralReleaseLevel { // case 3 -> release some collateral
			trnsfr = &types.Transfer{
				Owner: evt.Party(),
				Size:  1,
				Type:  types.TransferType_MARGIN_HIGH,
				Amount: &types.FinancialAmount{
					Asset:     evt.Asset(),
					Amount:    curMargin - margins.InitialMargin,
					MinAmount: 0,
				},
			}
		}

		// propage margins to the buffers
		e.marginsLevelsBuf.Add(*margins)

		risk := &marginChange{
			Margin:   evt,
			amount:   trnsfr.Amount.Amount,
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
) (okMargins []events.Margin, distressedPositions []events.MarketPosition) {
	okMargins = make([]events.Margin, 0, len(evts)/2)
	distressedPositions = make([]events.MarketPosition, 0, len(evts)/2)
	for _, evt := range evts {
		margins := e.calculateMargins(evt, int64(markPrice), *e.factors.RiskFactors[evt.Asset()], false)
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

		curMargin := int64(evt.MarginBalance())
		if curMargin > margins.MaintenanceMargin {
			okMargins = append(okMargins, evt)
		} else {
			distressedPositions = append(distressedPositions, evt)
		}
	}

	return
}

func (m marginChange) Amount() int64 {
	return m.amount
}

// Transfer - it's actually part of the embedded interface already, but we have to mask it, because this type contains another transfer
func (m marginChange) Transfer() *types.Transfer {
	return m.transfer
}

func (m marginChange) MarginLevels() *types.MarginLevels {
	return m.margins
}
