package supplied

import (
	"errors"
	"math"

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/shopspring/decimal"
)

// ErrNoValidOrders informs that there weren't any valid orders to cover the liquidity obligation with.
// This could happen when for a given side (buy or sell) limit orders don't supply enough liquidity and there aren't any
// valid pegged orders (all the prives are invalid) to cover it with.
var (
	ErrNoValidOrders = errors.New("no valid orders to cover the liquidity obligation with")
)

const (
	defaultInRangeProbabilityOfTrading = .5
	defaultMinimumProbabilityOfTrading = 1e-8
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
	ProbabilityOfTrading(currentPrice, yearFraction, orderPrice float64, isBid bool, applyMinMax bool, minPrice float64, maxPrice float64) float64
	GetProjectionHorizon() float64
}

// PriceMonitor provides the range of valid prices, that is prices that wouldn't trade the current trading mode
//go:generate go run github.com/golang/mock/mockgen -destination mocks/price_monitor_mock.go -package mocks code.vegaprotocol.io/vega/liquidity/supplied PriceMonitor
type PriceMonitor interface {
	GetValidPriceRange() (float64, float64)
}

// Engine provides functionality related to supplied liquidity
type Engine struct {
	rm RiskModel
	pm PriceMonitor

	horizon                        float64 // projection horizon used in probability calculations
	probabilityOfTradingTauScaling float64
	minProbabilityOfTrading        float64

	cachedMin float64
	cachedMax float64
	// Buy side cache
	bCache map[num.Uint]float64
	// Sell side cache
	sCache map[num.Uint]float64
}

// NewEngine returns a reference to a new supplied liquidity calculation engine
func NewEngine(riskModel RiskModel, priceMonitor PriceMonitor) *Engine {
	return &Engine{
		rm: riskModel,
		pm: priceMonitor,

		horizon:                        riskModel.GetProjectionHorizon(),
		probabilityOfTradingTauScaling: 1, // this is the same as the default in the netparams
		minProbabilityOfTrading:        defaultMinimumProbabilityOfTrading,
		bCache:                         map[num.Uint]float64{},
		sCache:                         map[num.Uint]float64{},
	}
}

func (e *Engine) OnMinProbabilityOfTradingLPOrdersUpdate(v float64) {
	e.minProbabilityOfTrading = v
}

func (e *Engine) OnProbabilityOfTradingTauScalingUpdate(v float64) {
	e.probabilityOfTradingTauScaling = v
}

// CalculateSuppliedLiquidity returns the current supplied liquidity per specified current mark price and order set
func (e *Engine) CalculateSuppliedLiquidity(
	bestBidPrice, bestAskPrice *num.Uint,
	orders []*types.Order,
) float64 {
	minPrice, maxPrice := e.pm.GetValidPriceRange()
	bLiq, sLiq := e.calculateBuySellLiquidityWithMinMax(bestBidPrice, bestAskPrice, orders, minPrice, maxPrice)
	return math.Min(bLiq, sLiq)
}

// CalculateLiquidityImpliedVolumes updates the LiquidityImpliedSize fields in LiquidityOrderReference so that the liquidity commitment is met.
// Current market price, liquidity obligation, and orders must be specified.
// Note that due to integer order size the actual liquidity provided will be more than or equal to the commitment amount.
func (e *Engine) CalculateLiquidityImpliedVolumes(
	bestBidPrice, bestAskPrice *num.Uint,
	liquidityObligation float64,
	orders []*types.Order,
	buyShapes, sellShapes []*LiquidityOrder,
) error {
	minPrice, maxPrice := e.pm.GetValidPriceRange()

	buySupplied, sellSupplied := e.calculateBuySellLiquidityWithMinMax(
		bestBidPrice, bestAskPrice, orders, minPrice, maxPrice)

	buyRemaining := liquidityObligation - buySupplied
	if err := e.updateSizes(buyRemaining, bestBidPrice, bestAskPrice, buyShapes, true, minPrice, maxPrice); err != nil {
		return err
	}

	sellRemaining := liquidityObligation - sellSupplied
	if err := e.updateSizes(sellRemaining, bestBidPrice, bestAskPrice, sellShapes, false, minPrice, maxPrice); err != nil {
		return err
	}

	return nil
}

// CalculateSuppliedLiquidity returns the current supplied liquidity per market specified in the constructor
func (e *Engine) calculateBuySellLiquidityWithMinMax(
	bestBidPrice, bestAskPrice *num.Uint,
	orders []*types.Order,
	minPrice, maxPrice float64,
) (float64, float64) {
	bLiq := 0.0
	sLiq := 0.0
	for _, o := range orders {
		if o.Side == types.Side_SIDE_BUY {
			// float64(o.Price.Uint64()) * float64(o.Remaining) * prob
			prob := e.getProbabilityOfTrading(bestBidPrice, bestAskPrice, o.Price, true, minPrice, maxPrice)
			d := decimal.NewFromFloat(prob)
			d = d.Mul(decimal.NewFromInt(int64(o.Remaining)))
			d = d.Mul(decimal.NewFromBigInt(o.Price.BigInt(), 0))
			liq, _ := d.Float64()
			bLiq += liq
		}
		if o.Side == types.Side_SIDE_SELL {
			// float64(o.Price.Uint64()) * float64(o.Remaining) * prob
			prob := e.getProbabilityOfTrading(bestBidPrice, bestAskPrice, o.Price, false, minPrice, maxPrice)
			d := decimal.NewFromFloat(prob)
			d = d.Mul(decimal.NewFromInt(int64(o.Remaining)))
			d = d.Mul(decimal.NewFromBigInt(o.Price.BigInt(), 0))
			liq, _ := d.Float64()
			sLiq += liq
		}
	}
	return bLiq, sLiq
}

func (e *Engine) updateSizes(
	liquidityObligation float64,
	bestBidPrice, bestAskprice *num.Uint,
	orders []*LiquidityOrder,
	isBid bool,
	minPrice, maxPrice float64,
) error {
	if liquidityObligation <= 0 {
		setSizesTo0(orders)
		return nil
	}

	var sum uint64 = 0
	probs := make([]float64, 0, len(orders))
	validatedProportions := make([]uint64, 0, len(orders))
	for _, o := range orders {
		proportion := o.Proportion

		prob := e.getProbabilityOfTrading(bestBidPrice, bestAskprice, o.Price, isBid, minPrice, maxPrice)
		if prob <= 0 {
			proportion = 0
		}

		sum += proportion
		validatedProportions = append(validatedProportions, proportion)
		probs = append(probs, prob)

	}
	if sum == 0 {
		return ErrNoValidOrders
	}
	fpSum := float64(sum)

	for i, o := range orders {
		scaling := 0.0
		prob := probs[i]
		if prob > 0 {
			fraction := float64(validatedProportions[i]) / fpSum
			scaling = fraction / prob
		}
		// uint64(math.Ceil(liquidityObligation * scaling / float64(o.Price.Uint64())))
		d := decimal.NewFromFloat(liquidityObligation * scaling)
		d = d.Div(decimal.NewFromBigInt(o.Price.BigInt(), 0)).Ceil()
		o.LiquidityImpliedVolume = uint64(d.BigInt().Int64())
	}
	return nil
}

func (e *Engine) getProbabilityOfTrading(bestBidPrice, bestAskPrice, orderPrice *num.Uint, isBid bool, minPrice float64, maxPrice float64) (f float64) {
	// if min, max changed since caches were created then reset
	if e.cachedMin != minPrice || e.cachedMax != maxPrice {
		e.bCache = make(map[num.Uint]float64, len(e.bCache))
		e.sCache = make(map[num.Uint]float64, len(e.sCache))
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
	currentPrice := bestAskPrice
	if orderPrice.LT(bestBidPrice) {
		currentPrice = bestBidPrice
	}

	prob := e.calcProbabilityOfTrading(currentPrice, orderPrice, isBid, minPrice, maxPrice)

	// if prob of trading is > than the minimum
	// we can return now.
	if prob >= e.minProbabilityOfTrading {
		return prob
	}

	// A failsafe to shift the probability of trading up to the minimum not to end up with unwieldy order sizes
	// This execution path should never be reached, but it is still theoretically possible for it to be reached due to rounding errors.
	return e.minProbabilityOfTrading
}

func (e *Engine) calcProbabilityOfTrading(currentPrice, orderPrice *num.Uint, isBid bool, minPrice float64, maxPrice float64) (f float64) {
	cache := e.sCache
	if isBid {
		cache = e.bCache
	}

	prob, ok := cache[*orderPrice]
	if !ok {
		tauScaled := e.horizon * e.probabilityOfTradingTauScaling
		// e.rm.ProbabilityOfTrading(float64(currentPrice.Uint64()), tauScaled, float64(orderPrice.Uint64()), isBid, true, minPrice, maxPrice)
		cp, _ := decimal.NewFromBigInt(currentPrice.BigInt(), 0).Float64()
		op, _ := decimal.NewFromBigInt(orderPrice.BigInt(), 0).Float64()
		prob = e.rm.ProbabilityOfTrading(cp, tauScaled, op, isBid, true, minPrice, maxPrice)
		cache[*orderPrice] = prob
	}
	return prob
}

func setSizesTo0(orders []*LiquidityOrder) {
	for _, o := range orders {
		o.LiquidityImpliedVolume = 0
	}
}
