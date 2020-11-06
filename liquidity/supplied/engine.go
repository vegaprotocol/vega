package supplied

import (
	"errors"
	"math"

	types "code.vegaprotocol.io/vega/proto"
)

var (
	// ErrNilOrderProvider signals that nil was supplied in place of OrderProvider
	ErrNilOrderProvider = errors.New("nil OrderProvider")
	// ErrNilRiskModel signals that nil was supplied in place of RiskModel
	ErrNilRiskModel = errors.New("nil RiskModel")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/order_provider_mock.go -package mocks code.vegaprotocol.io/vega/liquidity/supplied OrderProvider
// OrderProvider allows getting an order by its ID
type OrderProvider interface {
	GetOrderByID(orderID string) (*types.Order, error)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/risk_model_mock.go -package mocks code.vegaprotocol.io/vega/liquidity/supplied RiskModel
// RiskModel allows calculation of min/max price range and a probability of trading.
type RiskModel interface {
	PriceRange() (float64, float64)
	ProbabilityOfTrading(price float64, isBid bool, applyMinMax bool, minPrice float64, maxPrice float64) float64
}

type Engine struct {
	op OrderProvider
	rm RiskModel
}

// NewEngine returns a reference to a new supplied liquidity calculation engine if all arguments get supplied (with non-nil values) and an error otherwise
func NewEngine(orderProvider OrderProvider, riskModel RiskModel) (*Engine, error) {
	if orderProvider == nil {
		return nil, ErrNilOrderProvider
	}
	if riskModel == nil {
		return nil, ErrNilRiskModel
	}

	return &Engine{
		op: orderProvider,
		rm: riskModel,
	}, nil
}

// CalculateSuppliedLiquidity returns the current supplied liquidity per market specified in the constructor
func (e Engine) CalculateSuppliedLiquidity(liquidityProvisions ...types.LiquidityProvision) (float64, error) {
	buys, sells, err := e.getLiquidityProvisionOrders(liquidityProvisions...)
	if err != nil {
		return 0, err
	}
	min, max := e.rm.PriceRange()
	bLiq := e.calculateInstantaneousLiquidity(buys, true, min, max)
	sLiq := e.calculateInstantaneousLiquidity(sells, false, min, max)

	return math.Min(bLiq, sLiq), nil
}

func (e Engine) calculateInstantaneousLiquidity(mp map[uint64]uint64, isBuySide bool, minPrice, maxPrice float64) float64 {
	liquidity := 0.0
	for price, volume := range mp {
		fpPrice := float64(price)
		prob := e.rm.ProbabilityOfTrading(fpPrice, isBuySide, true, minPrice, maxPrice)

		liquidity += fpPrice * float64(volume) * prob
	}
	return liquidity
}

func (e Engine) getLiquidityProvisionOrders(liquidityProvisions ...types.LiquidityProvision) (map[uint64]uint64, map[uint64]uint64, error) {

	buys := make(map[uint64]uint64, len(liquidityProvisions))
	sells := make(map[uint64]uint64, len(liquidityProvisions))
	for _, lp := range liquidityProvisions {
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
