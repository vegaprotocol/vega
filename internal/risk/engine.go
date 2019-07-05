package risk

import (
	"context"
	"fmt"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/internal/events"

	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/positions"
	"code.vegaprotocol.io/vega/internal/vegatime"
	types "code.vegaprotocol.io/vega/proto"
)

type marginChange struct {
	events.Margin       // previous event that caused this change
	amount        int64 // the amount we need to move (positive is move to margin, neg == move to general)
	transfer      *types.Transfer
}

type Engine struct {
	Config
	log     *logging.Logger
	cfgMu   sync.Mutex
	model   Model
	factors *types.RiskResult
	waiting bool
	mu      sync.Mutex
}

func NewEngine(
	log *logging.Logger,
	config Config,
	model Model,
	initialFactors *types.RiskResult,
) *Engine {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	return &Engine{
		log:     log,
		Config:  config,
		factors: initialFactors,
		model:   model,
		waiting: false,
	}
}

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

func (re *Engine) CalculateFactors(now time.Time) {

	// don't calculate risk factors if we are before or at the next update time (calcs are before
	// processing and we calc factors after the time so we wait for time > nextUpdateTime) OR if we are
	// already waiting on risk calcs
	// NB: for continuous risk calcs nextUpdateTime will be 0 so we will always find time > nextUpdateTime
	if re.waiting || now.Before(vegatime.UnixNano(re.factors.NextUpdateTimestamp)) {
		return
	}

	wasCalculated, result := re.model.CalculateRiskFactors(re.factors)
	if wasCalculated {
		re.waiting = false
		if re.model.CalculationInterval() > 0 {
			result.NextUpdateTimestamp = now.Add(re.model.CalculationInterval()).UnixNano()
		}
		re.UpdateFactors(result)
	} else {
		re.waiting = true
	}
}

func (re *Engine) UpdatePositions(markPrice uint64, positions []positions.MarketPosition) {
	// todo(cdm): fix mark price overflow problems
	// todo(cdm): return action to possibly return action to update margin elsewhere rather than direct

	re.mu.Lock()
	for _, pos := range positions {
		notional := int64(markPrice) * pos.Size()
		for assetId, factor := range re.factors.RiskFactors {
			if pos.Size() > 0 {
				pos.UpdateMargin(assetId, uint64(float64(abs(notional))*factor.Long))
			} else {
				pos.UpdateMargin(assetId, uint64(float64(abs(notional))*factor.Short))
			}

			re.cfgMu.Lock()
			if re.LogMarginUpdate {
				re.log.Debug("Margins updated for position",
					logging.String("position", fmt.Sprintf("%+v", pos)))
			}
			re.cfgMu.Unlock()
		}
	}
	re.mu.Unlock()
}

// mock implementation, this wil return adjustments based on position updates
func (re *Engine) UpdateMargins(ctx context.Context, ch <-chan events.Margin, markPrice uint64) []events.Risk {
	re.mu.Lock()
	// we can allocate the return value here already
	// problem is that we don't know whether loss indicates a long/short position
	// @TODO ^^ Positions should provide this information, so we can pass this through correctly
	factors := map[string]*marginAmount{}
	ret := make([]events.Risk, 0, cap(ch))
	var err error
	// this will keep going until we've closed this channel
	// this can be the result of an error, or being "finished"
	for {
		select {
		case <-ctx.Done():
			// micro-optimisation perhaps, but hey... it's easy
			re.mu.Unlock()
			// this allows us to cancel in case of an error
			// we're not returning anything, because things didn't go as expected
			return nil
		case change, ok := <-ch:
			// channel is closed, and we've got a nil interface
			if !ok && change == nil {
				re.mu.Unlock()
				return ret
			}
			// just read from channel - this is the open position
			size := change.Size()
			// closed out, shouldn't be on this channel in the first place
			// but it's better to check anyway
			if size == 0 {
				continue
			}
			asset := change.Asset()
			factor, ok := factors[asset]
			if !ok {
				factor, err = re.getMargins(asset)
				if err != nil {
					// not sure what to do about these
					// @TODO this is debug for now, until we've got the asset format sorted out
					re.log.Debug(
						"No factor found for asset",
						logging.String("asset", asset),
					)
					continue
				}
				factors[asset] = factor
			}
			risk := factor.getChange(change, markPrice)
			if risk == nil {
				continue
			}
			ret = append(ret, risk)
		}
	}
}

func (re *Engine) UpdateFactors(result *types.RiskResult) {
	re.mu.Lock()
	re.factors = result
	re.mu.Unlock()
}

func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

func (m marginChange) Amount() int64 {
	return m.amount
}

// Transfer - it's actually part of the embedded interface already, but we have to mask it, because this type contains another transfer
func (m marginChange) Transfer() *types.Transfer {
	return m.transfer
}
