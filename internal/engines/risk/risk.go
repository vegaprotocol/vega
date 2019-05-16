package risk

import (
	"context"
	"fmt"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/internal/engines/events"
	"code.vegaprotocol.io/vega/internal/engines/position"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/riskmodels"
	"code.vegaprotocol.io/vega/internal/vegatime"
	types "code.vegaprotocol.io/vega/proto"
)

type marginChange struct {
	prev   events.MarginChange // previous event that caused this change
	amount int64               // the amount we need to move (positive is move to margin, neg == move to general)
}

type Engine struct {
	Config
	log     *logging.Logger
	cfgMu   sync.Mutex
	model   riskmodels.Model
	factors *types.RiskResult
	waiting bool
	mu      sync.Mutex // protect against factors beeing update while beein iterated over
}

func New(
	log *logging.Logger,
	config Config,
	model riskmodels.Model,
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

func (re *Engine) UpdatePositions(markPrice uint64, positions []position.MarketPosition) {
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
func (re *Engine) UpdateMarings(ctx context.Context, ch <-chan events.MarginChange, markPrice uint64) []interface{} {
	re.mu.Lock()
	defer re.mu.Unlock()
	// get config value up front
	re.cfgMu.Lock()
	logUpdate := re.LogMarginUpdate
	re.cfgMu.Unlock()
	// we can allocate the return value here already
	// problem is that we don't know whether loss indicates a long/short position
	// @TODO ^^ Positions should provide this information, so we can pass this through correctly
	ret := make([]*marginChange, 0, cap(ch))
	// this will keep going until we've closed this channel
	// this can be the result of an error, or being "finished"
	for {
		select {
		case <-ctx.Done():
			// this allows us to cancel in case of an error
			// we're not returning anything, because things didn't go as expected
			return nil
		case change := <-ch:
			// just read from channel
			size := change.Size()
			notional := int64(markPrice) * size
			factor, ok := re.factors.RiskFactors[change.Asset()]
			if !ok {
				// not sure what to do about these
				re.log.Warn(
					"No factor found for asset",
					logging.String("assetId", change.Asset()),
				)
				continue
			}
			var reqMargin uint64
			if size > 0 {
				reqMargin = uint64(float64(abs(notional)) * factor.Long)
			} else {
				reqMargin = uint64(float64(abs(notional)) * factor.Short)
			}
			// this is a bit silly here
			if logUpdate {
				re.log.Info("Margins updated for position",
					logging.String("position", fmt.Sprintf("%+v", change)))
			}
			marginBal := change.MarginBalance()
			if marginBal == reqMargin {
				continue
			}
			if marginBal < reqMargin {
				ret = append(ret, &marginChange{
					prev:   change,
					amount: int64(reqMargin),
				})
			} else {
				// delta, the bit we can move back
				ret = append(ret, &marginChange{
					prev:   change,
					amount: int64(marginBal) - int64(reqMargin),
				})
			}
		}
	}
	// just quick hack for return type
	intRet := make([]interface{}, 0, len(ret))
	for _, r := range ret {
		intRet = append(intRet, interface{}(r))
	}
	return intRet
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
