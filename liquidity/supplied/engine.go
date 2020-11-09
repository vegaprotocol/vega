package supplied

import (
	"math"

	types "code.vegaprotocol.io/vega/proto"
)

type LiquidityOrder struct {
	Price      uint64
	Proportion uint64

	NormalisedFraction   float64
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
	bLiq := 0.0
	sLiq := 0.0
	min, max := e.rp.ValidPriceRange()
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
				prob = e.rm.ProbabilityOfTrading(fpPrice, true, true, min, max)
				bProbs[price] = prob
			}
			bLiq += fpPrice * float64(volume) * prob
		}
		if o.Side == types.Side_SIDE_SELL {
			if prob, ok = sProbs[price]; !ok {
				prob = e.rm.ProbabilityOfTrading(fpPrice, false, true, min, max)
				sProbs[price] = prob
			}
			sLiq += fpPrice * float64(volume) * prob
		}
	}
	return math.Min(bLiq, sLiq), nil
}

// CalculateLiquidityImpliedSizes updates the LiquidityImpliedSize fields in LiquidityOrderReference so that the liquidity commitment is met.
// Note that due to integer order size the actual liquidity provided will be more than or equal to the commitment amount.
func (e Engine) CalculateLiquidityImpliedSizes(liquidityObligation float64, buyOrders []*LiquidityOrder, sellOrders []*LiquidityOrder) error {
	//get normalised fractions
	//get probability of trading
	updateNormalisedFractions(buyOrders)
	updateNormalisedFractions(sellOrders)
	min, max := e.rp.ValidPriceRange()
	e.updateSizes(liquidityObligation, buyOrders, true, min, max)
	e.updateSizes(liquidityObligation, sellOrders, false, min, max)

	return nil
}

func (e Engine) updateSizes(liquidityObligation float64, orders []*LiquidityOrder, buys bool, minPrice, maxPrice float64) {
	for _, o := range orders {
		prob := e.rm.ProbabilityOfTrading(float64(o.Price), buys, true, minPrice, maxPrice)
		o.LiquidityImpliedSize = uint64(math.Ceil(liquidityObligation * o.NormalisedFraction / prob))
	}
}

// TODO: This should be moved elsewhere an only called when Proportion in any of the LiquidityOrders changes
func updateNormalisedFractions(orders []*LiquidityOrder) {

	var sum uint64 = 0
	for _, o := range orders {
		sum += o.Proportion
	}
	fpSum := float64(sum)

	for _, o := range orders {
		o.NormalisedFraction = float64(o.Proportion) / fpSum
	}
}
