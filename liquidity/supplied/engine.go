package supplied

import (
	"errors"
	"math"

	types "code.vegaprotocol.io/vega/proto"
)

// ErrNoValidOrders informs that there weren't any valid orders to cover the liquidity obligation with.
// This could happen when for a given side (buy or sell) limit orders don't supply enough liquidity and there aren't any
// valid pegged orders (all the prives are invalid) to cover it with.
var (
	ErrNoValidOrders = errors.New("no valid orders to cover the liquidity obligation with")
)

// LiquidityOrder contains information required to compute volume required to fullfil liquidity obligation per set of liquidity provision orders for one side of the order book
type LiquidityOrder struct {
	OrderID string

	Price      uint64
	Proportion uint64

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

	horizon   float64 // projection horizon used in probability calculations
	cachedMin float64
	cachedMax float64
	bCache    map[uint64]float64
	sCache    map[uint64]float64
}

// NewEngine returns a reference to a new supplied liquidity calculation engine
func NewEngine(riskModel RiskModel, priceMonitor PriceMonitor) *Engine {
	return &Engine{
		rm: riskModel,
		pm: priceMonitor,

		horizon: riskModel.GetProjectionHorizon(),
		bCache:  map[uint64]float64{},
		sCache:  map[uint64]float64{},
	}
}

// CalculateSuppliedLiquidity returns the current supplied liquidity per specified current mark price and order set
func (e *Engine) CalculateSuppliedLiquidity(markPrice float64, orders []*types.Order) (float64, error) {
	minPrice, maxPrice := e.pm.GetValidPriceRange()
	bLiq, sLiq, err := e.calculateBuySellLiquidityWithMinMax(markPrice, orders, minPrice, maxPrice)
	if err != nil {
		return 0, err
	}
	return math.Min(bLiq, sLiq), nil
}

// CalculateLiquidityImpliedVolumes updates the LiquidityImpliedSize fields in LiquidityOrderReference so that the liquidity commitment is met.
// Current markt price, liquidity obligation, and orders must be specified.
// Note that due to integer order size the actual liquidity provided will be more than or equal to the commitment amount.
func (e *Engine) CalculateLiquidityImpliedVolumes(markPrice, liquidityObligation float64, orders []*types.Order, buyShapes []*LiquidityOrder, sellShapes []*LiquidityOrder) error {
	minPrice, maxPrice := e.pm.GetValidPriceRange()

	buySupplied, sellSupplied, err := e.calculateBuySellLiquidityWithMinMax(markPrice, orders, minPrice, maxPrice)
	if err != nil {
		return err
	}

	buyRemaining := liquidityObligation - buySupplied
	if err := e.updateSizes(buyRemaining, markPrice, buyShapes, true, minPrice, maxPrice); err != nil {
		return err
	}

	sellRemaining := liquidityObligation - sellSupplied
	if err := e.updateSizes(sellRemaining, markPrice, sellShapes, false, minPrice, maxPrice); err != nil {
		return err
	}

	return nil
}

// CalculateSuppliedLiquidity returns the current supplied liquidity per market specified in the constructor
func (e *Engine) calculateBuySellLiquidityWithMinMax(currentPrice float64, orders []*types.Order, minPrice, maxPrice float64) (float64, float64, error) {
	bLiq := 0.0
	sLiq := 0.0
	for _, o := range orders {
		if o.Side == types.Side_SIDE_BUY {
			bLiq += float64(o.Price) * float64(o.Remaining) * e.getProbabilityOfTrading(currentPrice, o.Price, true, minPrice, maxPrice)
		}
		if o.Side == types.Side_SIDE_SELL {
			sLiq += float64(o.Price) * float64(o.Remaining) * e.getProbabilityOfTrading(currentPrice, o.Price, false, minPrice, maxPrice)
		}
	}
	return bLiq, sLiq, nil
}

func (e *Engine) updateSizes(liquidityObligation, currentPrice float64, orders []*LiquidityOrder, isBid bool, minPrice, maxPrice float64) error {
	if liquidityObligation <= 0 {
		setSizesTo0(orders)
		return nil
	}

	var sum uint64 = 0
	probs := make([]float64, 0, len(orders))
	validatedProportions := make([]uint64, 0, len(orders))
	for _, o := range orders {
		proportion := o.Proportion

		prob := e.getProbabilityOfTrading(currentPrice, o.Price, isBid, minPrice, maxPrice)
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
		o.LiquidityImpliedVolume = uint64(math.Ceil(liquidityObligation * scaling / float64(o.Price)))
	}
	return nil
}

func (e *Engine) getProbabilityOfTrading(currentPrice float64, orderPrice uint64, isBid bool, minPrice float64, maxPrice float64) float64 {
	// if min, max changed since caches were created then reset
	if e.cachedMin != minPrice || e.cachedMax != maxPrice {
		e.bCache = make(map[uint64]float64, len(e.bCache))
		e.sCache = make(map[uint64]float64, len(e.sCache))
		e.cachedMin, e.cachedMax = minPrice, maxPrice
	}

	cache := e.sCache
	if isBid {
		cache = e.bCache
	}

	if _, ok := cache[orderPrice]; !ok {
		prob := e.rm.ProbabilityOfTrading(currentPrice, e.horizon, float64(orderPrice), isBid, true, minPrice, maxPrice)
		cache[orderPrice] = prob
	}
	return cache[orderPrice]
}

func setSizesTo0(orders []*LiquidityOrder) {
	for _, o := range orders {
		o.LiquidityImpliedVolume = 0
	}
}
