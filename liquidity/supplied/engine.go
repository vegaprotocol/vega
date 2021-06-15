package supplied

import (
	"errors"

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

// ErrNoValidOrders informs that there weren't any valid orders to cover the liquidity obligation with.
// This could happen when for a given side (buy or sell) limit orders don't supply enough liquidity and there aren't any
// valid pegged orders (all the prives are invalid) to cover it with.
var (
	ErrNoValidOrders = errors.New("no valid orders to cover the liquidity obligation with")
)

var (
	defaultInRangeProbabilityOfTrading = num.DecimalFromFloat(0.5)
	defaultMinimumProbabilityOfTrading = num.DecimalFromFloat(1e-8)
)

// LiquidityOrder contains information required to compute volume required to fullfil liquidity obligation per set of liquidity provision orders for one side of the order book
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
	ProbabilityOfTrading(currentPrice, orderPrice, minPrice, maxPrice *num.Uint, yearFraction num.Decimal, isBid, applyMinMax bool) num.Decimal
	GetProjectionHorizon() num.Decimal
}

// PriceMonitor provides the range of valid prices, that is prices that wouldn't trade the current trading mode
//go:generate go run github.com/golang/mock/mockgen -destination mocks/price_monitor_mock.go -package mocks code.vegaprotocol.io/vega/liquidity/supplied PriceMonitor
type PriceMonitor interface {
	GetValidPriceRange() (*num.Uint, *num.Uint)
}

// Engine provides functionality related to supplied liquidity
type Engine struct {
	rm RiskModel
	pm PriceMonitor

	horizon                        num.Decimal // projection horizon used in probability calculations
	probabilityOfTradingTauScaling num.Decimal
	minProbabilityOfTrading        num.Decimal

	cachedMin *num.Uint
	cachedMax *num.Uint
	// Buy side cache
	bCache map[num.Uint]num.Decimal
	// Sell side cache
	sCache map[num.Uint]num.Decimal
}

// NewEngine returns a reference to a new supplied liquidity calculation engine
func NewEngine(riskModel RiskModel, priceMonitor PriceMonitor) *Engine {
	return &Engine{
		rm: riskModel,
		pm: priceMonitor,

		horizon:                        riskModel.GetProjectionHorizon(),
		probabilityOfTradingTauScaling: num.DecimalFromFloat(1.0), // this is the same as the default in the netparams
		minProbabilityOfTrading:        defaultMinimumProbabilityOfTrading,
		bCache:                         map[num.Uint]num.Decimal{},
		sCache:                         map[num.Uint]num.Decimal{},
	}
}

func (e *Engine) OnMinProbabilityOfTradingLPOrdersUpdate(v num.Decimal) {
	e.minProbabilityOfTrading = v
}

func (e *Engine) OnProbabilityOfTradingTauScalingUpdate(v num.Decimal) {
	e.probabilityOfTradingTauScaling = v
}

// CalculateSuppliedLiquidity returns the current supplied liquidity per specified current mark price and order set
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
	if err := e.updateSizes(buyRemaining, bestBidPrice.Clone(), bestAskPrice.Clone(), buyShapes, true, minPrice.Clone(), maxPrice.Clone()); err != nil {
		return err
	}

	sellRemaining := liquidityObligation.Clone()
	sellRemaining.Sub(sellRemaining, sellSupplied)
	if err := e.updateSizes(sellRemaining, bestBidPrice.Clone(), bestAskPrice.Clone(), sellShapes, false, minPrice.Clone(), maxPrice.Clone()); err != nil {
		return err
	}

	return nil
}

// CalculateSuppliedLiquidity returns the current supplied liquidity per market specified in the constructor
func (e *Engine) calculateBuySellLiquidityWithMinMax(
	bestBidPrice, bestAskPrice *num.Uint,
	orders []*types.Order,
	minPrice, maxPrice *num.Uint,
) (*num.Uint, *num.Uint) {
	bLiq := num.DecimalFromFloat(0.0)
	sLiq := num.DecimalFromFloat(0.0)
	for _, o := range orders {
		if o.Side == types.Side_SIDE_BUY {
			// float64(o.Price.Uint64()) * float64(o.Remaining) * prob
			prob := e.getProbabilityOfTrading(bestBidPrice.Clone(), bestAskPrice.Clone(), o.Price.Clone(), true, minPrice.Clone(), maxPrice.Clone())
			d := prob.Mul(num.DecimalFromUint(num.NewUint(o.Remaining)))
			d = d.Mul(num.DecimalFromUint(o.Price))
			bLiq = bLiq.Add(d)
		}
		if o.Side == types.Side_SIDE_SELL {
			// float64(o.Price.Uint64()) * float64(o.Remaining) * prob
			prob := e.getProbabilityOfTrading(bestBidPrice.Clone(), bestAskPrice.Clone(), o.Price.Clone(), false, minPrice.Clone(), maxPrice.Clone())
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
	minPrice, maxPrice *num.Uint,
) error {
	if liquidityObligation.IsZero() || liquidityObligation.IsNegative() {
		setSizesTo0(orders)
		return nil
	}

	sum := num.DecimalFromFloat(0.0)
	probs := make([]num.Decimal, 0, len(orders))
	validatedProportions := make([]num.Decimal, 0, len(orders))
	for _, o := range orders {
		proportion := num.DecimalFromUint(num.NewUint(o.Proportion))

		prob := e.getProbabilityOfTrading(bestBidPrice.Clone(), bestAskprice.Clone(), o.Price.Clone(), isBid, minPrice.Clone(), maxPrice.Clone())
		if prob.LessThanOrEqual(num.DecimalFromFloat(0.0)) {
			proportion = num.DecimalFromFloat(0.0)
		}

		sum = sum.Add(proportion)
		validatedProportions = append(validatedProportions, proportion)
		probs = append(probs, prob)
	}
	if sum.IsZero() {
		return ErrNoValidOrders
	}

	for i, o := range orders {
		scaling := num.DecimalFromFloat(0.0)
		prob := probs[i]
		if prob.GreaterThan(num.DecimalFromFloat(0.0)) {
			fraction := validatedProportions[i].Div(sum)
			scaling = fraction.Div(prob)
		}
		// uint64(math.Ceil(liquidityObligation * scaling / float64(o.Price.Uint64())))
		d := num.DecimalFromUint(liquidityObligation)
		d = d.Mul(scaling)
		d = d.Div(num.DecimalFromUint(o.Price)).Ceil()
		o.LiquidityImpliedVolume = uint64(d.IntPart())
	}
	return nil
}

func (e *Engine) getProbabilityOfTrading(bestBidPrice, bestAskPrice, orderPrice *num.Uint, isBid bool, minPrice *num.Uint, maxPrice *num.Uint) num.Decimal {
	// if min, max changed since caches were created then reset
	if e.cachedMin != minPrice || e.cachedMax != maxPrice {
		e.bCache = make(map[num.Uint]num.Decimal, len(e.bCache))
		e.sCache = make(map[num.Uint]num.Decimal, len(e.sCache))
		e.cachedMin, e.cachedMax = minPrice, maxPrice
	}

	// Any part of shape that's pegged between or equal to
	// best_static_bid and best_static_ask
	// has probability of trading = 1.
	if orderPrice.GTE(bestBidPrice) && orderPrice.LTE(bestAskPrice) {
		return defaultInRangeProbabilityOfTrading
	}

	// Any part of shape that the peg puts at lower price
	// than best bid will use probability of trading computed
	// from best_static_bid to calculate volume.
	// Any part of shape that the peg puts at price above
	// best_static_ask will use probability of trading computed
	// from best_static_ask.
	var currentPrice *num.Uint
	if orderPrice.LT(bestBidPrice) {
		currentPrice = bestBidPrice.Clone()
	} else {
		currentPrice = bestAskPrice.Clone()
	}

	prob := e.calcProbabilityOfTrading(currentPrice, orderPrice, isBid, minPrice, maxPrice)

	// if prob of trading is > than the minimum
	// we can return now.
	if prob.GreaterThanOrEqual(e.minProbabilityOfTrading) {
		return prob
	}

	// A failsafe to shift the probability of trading up to the minimum not to end up with unwieldy order sizes
	// This execution path should never be reached, but it is still theoretically possible for it to be reached due to rounding errors.
	return e.minProbabilityOfTrading
}

func (e *Engine) calcProbabilityOfTrading(currentPrice, orderPrice *num.Uint, isBid bool, minPrice, maxPrice *num.Uint) num.Decimal {
	cache := e.sCache
	if isBid {
		cache = e.bCache
	}

	prob, ok := cache[*orderPrice]
	if !ok {
		tauScaled := e.horizon.Mul(e.probabilityOfTradingTauScaling)
		prob = e.rm.ProbabilityOfTrading(currentPrice, orderPrice, minPrice, maxPrice, tauScaled, isBid, true)
		cache[*orderPrice] = prob
	}
	return prob
}

func setSizesTo0(orders []*LiquidityOrder) {
	for _, o := range orders {
		o.LiquidityImpliedVolume = 0
	}
}
