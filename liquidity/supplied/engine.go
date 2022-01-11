package supplied

import (
	"context"
	"errors"

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"code.vegaprotocol.io/vega/types/statevar"
)

// ErrNoValidOrders informs that there weren't any valid orders to cover the liquidity obligation with.
// This could happen when for a given side (buy or sell) limit orders don't supply enough liquidity and there aren't any
// valid pegged orders (all the prives are invalid) to cover it with.
var (
	ErrNoValidOrders = errors.New("no valid orders to cover the liquidity obligation with")
)

// LiquidityOrder contains information required to compute volume required to fullfil liquidity obligation per set of liquidity provision orders for one side of the order book.
type LiquidityOrder struct {
	OrderID string

	Price      *num.Uint
	Proportion uint64
	Peg        *types.PeggedOrder

	LiquidityImpliedVolume uint64
}

// RiskModel allows calculation of min/max price range and a probability of trading.
//go:generate go run github.com/golang/mock/mockgen -destination mocks/risk_model_mock.go -package mocks code.vegaprotocol.io/vega/liquidity/supplied RiskModel
type RiskModel interface {
	ProbabilityOfTrading(currentPrice, orderPrice *num.Uint, minPrice, maxPrice num.Decimal, yearFraction num.Decimal, isBid, applyMinMax bool) num.Decimal
	GetProjectionHorizon() num.Decimal
}

// PriceMonitor provides the range of valid prices, that is prices that wouldn't trade the current trading mode
//go:generate go run github.com/golang/mock/mockgen -destination mocks/price_monitor_mock.go -package mocks code.vegaprotocol.io/vega/liquidity/supplied PriceMonitor
type PriceMonitor interface {
	GetValidPriceRange() (num.WrappedDecimal, num.WrappedDecimal)
}

type StateVarEngine interface {
	AddStateVariable(asset, market string, converter statevar.Converter, startCalculation func(string, statevar.FinaliseCalculation), trigger []statevar.StateVarEventType, result func(context.Context, statevar.StateVariableResult) error) error
}

// Engine provides functionality related to supplied liquidity.
type Engine struct {
	rm                             RiskModel
	pm                             PriceMonitor
	marketID                       string
	horizon                        num.Decimal // projection horizon used in probability calculations
	probabilityOfTradingTauScaling num.Decimal
	minProbabilityOfTrading        num.Decimal

	cachedMin *num.Uint
	cachedMax *num.Uint
	// Bid cache
	bCache map[num.Uint]num.Decimal
	// Ask cache
	aCache  map[num.Uint]num.Decimal
	changed bool

	pot                 *probabilityOfTrading
	getBestStaticPrices func() (*num.Uint, *num.Uint, error)
}

// NewEngine returns a reference to a new supplied liquidity calculation engine.
func NewEngine(riskModel RiskModel, priceMonitor PriceMonitor, asset, marketID string, stateVarEngine StateVarEngine) *Engine {
	e := &Engine{
		rm:                             riskModel,
		pm:                             priceMonitor,
		marketID:                       marketID,
		cachedMin:                      num.Zero(),
		cachedMax:                      num.Zero(),
		horizon:                        riskModel.GetProjectionHorizon(),
		probabilityOfTradingTauScaling: num.DecimalFromInt64(1), // this is the same as the default in the netparams
		minProbabilityOfTrading:        defaultMinimumProbabilityOfTrading,
		bCache:                         map[num.Uint]num.Decimal{},
		aCache:                         map[num.Uint]num.Decimal{},
		changed:                        true,
		pot:                            &probabilityOfTrading{},
	}

	stateVarEngine.AddStateVariable(asset, marketID, probabilityOfTradingConverter{}, e.startCalcProbOfTrading, []statevar.StateVarEventType{statevar.StateVarEventTypeTimeTrigger, statevar.StateVarEventTypeAuctionEnded, statevar.StateVarEventTypeOpeningAuctionFirstUncrossingPrice}, e.updateProbabilities)
	return e
}

func (e *Engine) SetGetStaticPricesFunc(f func() (*num.Uint, *num.Uint, error)) {
	e.getBestStaticPrices = f
}

func (e *Engine) OnMinProbabilityOfTradingLPOrdersUpdate(v num.Decimal) {
	e.minProbabilityOfTrading = v
}

func (e *Engine) OnProbabilityOfTradingTauScalingUpdate(v num.Decimal) {
	e.probabilityOfTradingTauScaling = v
}

// CalculateSuppliedLiquidity returns the current supplied liquidity per specified current mark price and order set.
func (e *Engine) CalculateSuppliedLiquidity(
	bestBidPrice, bestAskPrice *num.Uint,
	orders []*types.Order,
) *num.Uint {
	// Update this to return *Uint as part of monitor refactor TODO UINT
	minPrice, maxPrice := e.pm.GetValidPriceRange()
	bLiq, sLiq := e.calculateBuySellLiquidityWithMinMax(bestBidPrice, bestAskPrice, orders, minPrice, maxPrice)

	return num.Min(bLiq, sLiq)
}

// CalculateLiquidityImpliedVolumes updates the LiquidityImpliedSize fields in LiquidityOrderReference so that the liquidity commitment is met.
// Current market price, liquidity obligation, and orders must be specified.
// Note that due to integer order size the actual liquidity provided will be more than or equal to the commitment amount.
func (e *Engine) CalculateLiquidityImpliedVolumes(
	bestBidPrice, bestAskPrice *num.Uint,
	liquidityObligation *num.Uint,
	orders []*types.Order,
	buyShapes, sellShapes []*LiquidityOrder,
) error {
	// Update this to return *Uint as part of monitor refactor PETE TODO
	minPrice, maxPrice := e.pm.GetValidPriceRange()

	buySupplied, sellSupplied := e.calculateBuySellLiquidityWithMinMax(
		bestBidPrice, bestAskPrice, orders, minPrice, maxPrice)

	buyRemaining := liquidityObligation.Clone()
	buyRemaining.Sub(buyRemaining, buySupplied)
	if err := e.updateSizes(buyRemaining, bestBidPrice.Clone(), bestAskPrice.Clone(), buyShapes, true, minPrice, maxPrice); err != nil {
		return err
	}

	sellRemaining := liquidityObligation.Clone()
	sellRemaining.Sub(sellRemaining, sellSupplied)
	if err := e.updateSizes(sellRemaining, bestBidPrice.Clone(), bestAskPrice.Clone(), sellShapes, false, minPrice, maxPrice); err != nil {
		return err
	}

	return nil
}

// calculateBuySellLiquidityWithMinMax returns the current supplied liquidity per market specified in the constructor.
func (e *Engine) calculateBuySellLiquidityWithMinMax(
	bestBidPrice, bestAskPrice *num.Uint,
	orders []*types.Order,
	minPrice, maxPrice num.WrappedDecimal,
) (*num.Uint, *num.Uint) {
	bLiq := num.DecimalZero()
	sLiq := num.DecimalZero()
	for _, o := range orders {
		if o.Side == types.SideBuy {
			// float64(o.Price.Uint64()) * float64(o.Remaining) * prob
			prob := getProbabilityOfTrading(e.getBestStaticPrices, e.pot, o.Price.ToDecimal(), true, e.minProbabilityOfTrading)
			d := prob.Mul(num.DecimalFromUint(num.NewUint(o.Remaining)))
			d = d.Mul(num.DecimalFromUint(o.Price))
			bLiq = bLiq.Add(d)
		}
		if o.Side == types.SideSell {
			// float64(o.Price.Uint64()) * float64(o.Remaining) * prob
			prob := getProbabilityOfTrading(e.getBestStaticPrices, e.pot, o.Price.ToDecimal(), false, e.minProbabilityOfTrading)
			d := prob.Mul(num.DecimalFromUint(num.NewUint(o.Remaining)))
			d = d.Mul(num.DecimalFromUint(o.Price))
			sLiq = sLiq.Add(d)
		}
	}
	bl, _ := num.UintFromDecimal(bLiq)
	sl, _ := num.UintFromDecimal(sLiq)
	return bl, sl
}

func (e *Engine) updateSizes(
	liquidityObligation *num.Uint,
	bestBidPrice, bestAskprice *num.Uint,
	orders []*LiquidityOrder,
	isBid bool,
	minPrice, maxPrice num.WrappedDecimal,
) error {
	if liquidityObligation.IsZero() || liquidityObligation.IsNegative() {
		setSizesTo0(orders)
		return nil
	}

	sum := num.DecimalZero()
	probs := make([]num.Decimal, 0, len(orders))
	validatedProportions := make([]num.Decimal, 0, len(orders))
	for _, o := range orders {
		proportion := num.DecimalFromUint(num.NewUint(o.Proportion))

		prob := getProbabilityOfTrading(e.getBestStaticPrices, e.pot, o.Price.ToDecimal(), isBid, e.minProbabilityOfTrading)
		if prob.IsZero() || prob.IsNegative() {
			proportion = num.DecimalZero()
		}

		sum = sum.Add(proportion)
		validatedProportions = append(validatedProportions, proportion)
		probs = append(probs, prob)
	}
	if sum.IsZero() {
		return ErrNoValidOrders
	}

	for i, o := range orders {
		scaling := num.DecimalZero()
		if prob := probs[i]; !prob.IsZero() {
			fraction := validatedProportions[i].Div(sum)
			scaling = fraction.Div(prob)
		}
		// uint64(math.Ceil(liquidityObligation * scaling / float64(o.Price.Uint64())))
		d := num.DecimalFromUint(liquidityObligation)
		d = d.Mul(scaling)
		liv, _ := num.UintFromDecimal(d.Div(num.DecimalFromUint(o.Price)).Ceil())
		o.LiquidityImpliedVolume = liv.Uint64()
	}
	return nil
}

func setSizesTo0(orders []*LiquidityOrder) {
	for _, o := range orders {
		o.LiquidityImpliedVolume = 0
	}
}
