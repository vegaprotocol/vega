package risk

import (
	"context"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"code.vegaprotocol.io/vega/types/statevar"
)

type RiskFactorConverter struct{}

var riskFactorTolerance = num.MustDecimalFromString("1e-6")

func (RiskFactorConverter) BundleToInterface(kvb *statevar.KeyValueBundle) statevar.StateVariableResult {
	return &types.RiskFactor{
		Short: kvb.KVT[0].Val.(*statevar.DecimalScalar).Val,
		Long:  kvb.KVT[1].Val.(*statevar.DecimalScalar).Val,
	}
}

func (RiskFactorConverter) InterfaceToBundle(res statevar.StateVariableResult) *statevar.KeyValueBundle {
	value := res.(*types.RiskFactor)
	return &statevar.KeyValueBundle{
		KVT: []statevar.KeyValueTol{
			{Key: "short", Val: &statevar.DecimalScalar{Val: value.Short}, Tolerance: riskFactorTolerance},
			{Key: "long", Val: &statevar.DecimalScalar{Val: value.Long}, Tolerance: riskFactorTolerance},
		},
	}
}

// startRiskFactorsCalculation kicks off the risk factors calculation, done asynchronously for illustration.
func (e *Engine) startRiskFactorsCalculation(eventID string, endOfCalcCallback statevar.FinaliseCalculation) {
	rf := e.model.CalculateRiskFactors()
	e.log.Info("risk factors calculated", logging.String("event-id", eventID), logging.Decimal("short", rf.Short), logging.Decimal("long", rf.Long))
	endOfCalcCallback.CalculationFinished(eventID, rf, nil)
}

// CalculateRiskFactorsForTest is a hack for testing for setting directly the risk factors for a market.
func (e *Engine) CalculateRiskFactorsForTest() {
	e.factors = e.model.CalculateRiskFactors()
	e.factors.Market = e.mktID
}

// updateRiskFactor sets the risk factor value to that of the decimal consensus value.
func (e *Engine) updateRiskFactor(ctx context.Context, res statevar.StateVariableResult) error {
	e.factors = res.(*types.RiskFactor)
	e.factors.Market = e.mktID
	e.riskFactorsInitialised = true
	e.log.Info("consensus reached for risk factors", logging.String("market", e.mktID), logging.Decimal("short", e.factors.Short), logging.Decimal("long", e.factors.Long))
	// then we can send in the broker
	e.broker.Send(events.NewRiskFactorEvent(ctx, *e.factors))
	return nil
}

func (e *Engine) IsRiskFactorInitialised() bool {
	return e.riskFactorsInitialised
}
