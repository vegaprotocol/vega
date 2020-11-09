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

type LiquidityOrder struct {
	Price      uint64
	Proportion uint64

	NormalisedFraction   float64
	LiquidityImpliedSize uint64
}

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

// UpdateLiquidityImpliedSizes updates the LiquidityImpliedSize fields in LiquidityOrderReference so that the liquidity commitment is met.
// Note that due to integer order size the actual liquidity provided will be more than or equal to the commitment amount.
func (e Engine) CalculateLiquidityImpliedSizes(liquidityObligation float64, buyOrders []*LiquidityOrder, sellOrders []*LiquidityOrder) error {
	//get normalised fractions
	//get probability of trading
	updateNormalisedFractions(buyOrders)
	updateNormalisedFractions(sellOrders)
	min, max := e.rm.PriceRange()
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
