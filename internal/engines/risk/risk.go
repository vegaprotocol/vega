package risk

import (
	"fmt"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/internal/engines/position"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/riskmodels"
	"code.vegaprotocol.io/vega/internal/vegatime"
	types "code.vegaprotocol.io/vega/proto"
)

type Engine struct {
	*Config

	model   riskmodels.Model
	factors *types.RiskResult
	waiting bool
	mu      sync.Mutex // protect against factors beeing update while beein iterated over
}

func New(
	config *Config,
	model riskmodels.Model,
	initialFactors *types.RiskResult,
) *Engine {
	return &Engine{
		Config:  config,
		factors: initialFactors,
		model:   model,
		waiting: false,
	}
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

			if re.LogMarginUpdate {
				re.log.Info("Margins updated for position",
					logging.String("position", fmt.Sprintf("%+v", pos)))
			}
		}
	}
	re.mu.Unlock()
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
