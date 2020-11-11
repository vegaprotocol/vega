package supplied

import (
	"math"

	types "code.vegaprotocol.io/vega/proto"
)

type LiquidityOrder struct {
	Price      uint64
	Proportion uint64

	LiquidityImpliedSize uint64
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/risk_model_mock.go -package mocks code.vegaprotocol.io/vega/liquidity/supplied RiskModel
// RiskModel allows calculation of min/max price range and a probability of trading.
type RiskModel interface {
	ProbabilityOfTrading(price float64, isBid bool, applyMinMax bool, minPrice float64, maxPrice float64) float64
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/valid_price_range_provider_mock.go -package mocks code.vegaprotocol.io/vega/liquidity/supplied ValidPriceRangeProvider
// ValidPriceRangeProvider provides the range of valid prices, that is prices that wouldn't trade the current trading mode
type ValidPriceRangeProvider interface {
	ValidPriceRange() (float64, float64)
}

type Engine struct {
	rm RiskModel
	rp ValidPriceRangeProvider
}

// NewEngine returns a reference to a new supplied liquidity calculation engine
func NewEngine(riskModel RiskModel, validPriceRangeProvider ValidPriceRangeProvider) *Engine {
	return &Engine{
		rm: riskModel,
		rp: validPriceRangeProvider,
	}
}

// CalculateSuppliedLiquidity returns the current supplied liquidity per market specified in the constructor
func (e Engine) CalculateSuppliedLiquidity(orders []types.Order) (float64, error) {
	minPrice, maxPrice := e.rp.ValidPriceRange()
	bLiq, sLiq, err := e.calculateBuySellLiquidityWithMinMax(orders, minPrice, maxPrice)
	if err != nil {
		return 0, err
	}
	return math.Min(bLiq, sLiq), nil
}

// CalculateLiquidityImpliedSizes updates the LiquidityImpliedSize fields in LiquidityOrderReference so that the liquidity commitment is met.
// Note that due to integer order size the actual liquidity provided will be more than or equal to the commitment amount.
func (e Engine) CalculateLiquidityImpliedSizes(liquidityObligation float64, buyLimitOrders []types.Order, sellLimitOrders []types.Order, buyShapes []*LiquidityOrder, sellShapes []*LiquidityOrder) error {
	minPrice, maxPrice := e.rp.ValidPriceRange()

	limitOrders := make([]types.Order, 0, len(buyLimitOrders)+len(sellLimitOrders))
	limitOrders = append(limitOrders, buyLimitOrders...)
	limitOrders = append(limitOrders, sellLimitOrders...)

	buySupplied, sellSupplied, err := e.calculateBuySellLiquidityWithMinMax(limitOrders, minPrice, maxPrice)
	if err != nil {
		return err
	}

	buyRemaining := liquidityObligation - buySupplied
	e.updateSizes(buyRemaining, buyShapes, true, minPrice, maxPrice)

	sellRemaining := liquidityObligation - sellSupplied
	e.updateSizes(sellRemaining, sellShapes, false, minPrice, maxPrice)
	return nil
}

// CalculateSuppliedLiquidity returns the current supplied liquidity per market specified in the constructor
func (e Engine) calculateBuySellLiquidityWithMinMax(orders []types.Order, minPrice, maxPrice float64) (float64, float64, error) {
	bLiq := 0.0
	sLiq := 0.0
	var bProbs map[uint64]float64 = make(map[uint64]float64)
	var sProbs map[uint64]float64 = make(map[uint64]float64)
	var prob float64
	var ok bool
	for _, o := range orders {
		price := o.Price
		fpPrice := float64(price)
		volume := o.Remaining

		if o.Side == types.Side_SIDE_BUY {
			if prob, ok = bProbs[price]; !ok {
				prob = e.rm.ProbabilityOfTrading(fpPrice, true, true, minPrice, maxPrice)
				bProbs[price] = prob
			}
			bLiq += fpPrice * float64(volume) * prob
		}
		if o.Side == types.Side_SIDE_SELL {
			if prob, ok = sProbs[price]; !ok {
				prob = e.rm.ProbabilityOfTrading(fpPrice, false, true, minPrice, maxPrice)
				sProbs[price] = prob
			}
			sLiq += fpPrice * float64(volume) * prob
		}
	}
	return bLiq, sLiq, nil
}

func (e Engine) updateSizes(liquidityObligation float64, orders []*LiquidityOrder, buys bool, minPrice, maxPrice float64) {
	if liquidityObligation <= 0 {
		setSizesTo0(orders)
		return
	}

	var sum uint64 = 0
	probs := make([]float64, 0, len(orders))
	validatedProportions := make([]uint64, 0, len(orders))
	for _, o := range orders {
		proportion := o.Proportion
		prob := e.rm.ProbabilityOfTrading(float64(o.Price), buys, true, minPrice, maxPrice)
		if prob <= 0 {
			proportion = 0
		}
		sum += proportion
		validatedProportions = append(validatedProportions, proportion)
		probs = append(probs, prob)

	}
	fpSum := float64(sum)

	for i, o := range orders {
		scaling := 0.0
		prob := probs[i]
		if prob > 0 {
			fraction := float64(validatedProportions[i]) / fpSum
			scaling = fraction / prob
		}
		o.LiquidityImpliedSize = uint64(math.Ceil(liquidityObligation * scaling / float64(o.Price)))
	}
}

func setSizesTo0(orders []*LiquidityOrder) {
	for _, o := range orders {
		o.LiquidityImpliedSize = 0
	}
}
