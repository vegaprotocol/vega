package supplied

import (
	"math"

	types "code.vegaprotocol.io/vega/proto"
)

type LiquidityProvisionProvider interface {
	GetLiquidityProvisions(market string) ([]types.LiquidityProvision, error)
}

type OrderProvider interface {
	GetOrderByID(orderID string) (*types.Order, error)
}

type ProbabilityOfTradingCalculator interface {
	ProbabilityOfTrading(price float64, isBid bool, applyMinMax bool, minPrice float64, maxPrice float64) float64
}

type PariceRangeProvider interface {
	PriceRange() (float64, float64)
}

type Engine struct {
	mId string

	lpp LiquidityProvisionProvider
	op  OrderProvider
	ptc ProbabilityOfTradingCalculator
	prp PariceRangeProvider

	//TODO: Move buys, sells here to aid memory usage
}

func (e Engine) GetSuppliedLiquidity() (float64, error) {
	buys, sells, err := e.getLiquidityProvisionOrders()
	if err != nil {
		return 0, err
	}
	bLiq := e.calculateInstantaneousLiquidity(buys, true)
	sLiq := e.calculateInstantaneousLiquidity(sells, false)

	return math.Min(bLiq, sLiq), nil
}

func (e Engine) calculateInstantaneousLiquidity(mp map[uint64]uint64, isBuySide bool) float64 {
	min, max := e.prp.PriceRange()
	liquidity := 0.0
	for price, volume := range mp {
		fpPrice := float64(price)
		prob := e.ptc.ProbabilityOfTrading(fpPrice, isBuySide, true, min, max)

		liquidity += fpPrice * float64(volume) * prob
	}
	return liquidity
}

func (e Engine) getLiquidityProvisionOrders() (map[uint64]uint64, map[uint64]uint64, error) {
	lps, err := e.lpp.GetLiquidityProvisions(e.mId)
	if err != nil {
		return nil, nil, err
	}

	buys := make(map[uint64]uint64, len(lps))
	sells := make(map[uint64]uint64, len(lps))
	for _, lp := range lps {
		if err := e.sumVolumePerPrice(buys, lp.Buys); err != nil {
			return nil, nil, err
		}
		if err := e.sumVolumePerPrice(sells, lp.Sells); err != nil {
			return nil, nil, err
		}
	}
	return buys, sells, nil
}

func (e Engine) sumVolumePerPrice(mp map[uint64]uint64, lors []*types.LiquidityOrderReference) error {
	for _, lor := range lors {
		order, err := e.op.GetOrderByID(lor.OrderID)
		if err != nil {
			return err
		}
		mp[order.Price] += order.Remaining
	}
	return nil
}

// TODO: Do we want to get min/max from model, or are LP orders already guaranteed to be in that range?
// TODO: Do we need a liquidity engine that liqudity service will reference? Then we could pass reference to that engine to market and use it here.
