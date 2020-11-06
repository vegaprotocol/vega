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

// UpdateLiquidityImpliedSizes updates the LiquidityImpliedSize fields in LiquidityOrderReference so that the liquidity commitment is met.
// Note that due to integer order size the actual liquidity provided will be more than or equal to the commitment amount.
func (e Engine) UpdateLiquidityImpliedSizes(liquidityObligation float64, liquidityProvision types.LiquidityProvision) error {
	//get normalised fractions
	//get probability of trading
	updateNormalisedFractions(liquidityProvision.Buys)
	updateNormalisedFractions(liquidityProvision.Sells)
	min, max := e.rm.PriceRange()
	err := e.updateSizes(liquidityObligation, liquidityProvision.Buys, true, min, max)
	if err != nil {
		return err
	}
	err = e.updateSizes(liquidityObligation, liquidityProvision.Sells, false, min, max)
	if err != nil {
		return err
	}
	return nil
}

func (e Engine) updateSizes(liquidityObligation float64, lors []*types.LiquidityOrderReference, isBuySide bool, minPrice, maxPrice float64) error {
	for _, lor := range lors {
		price, err := e.getPrice(*lor)
		if err != nil {
			return err
		}
		prob := e.rm.ProbabilityOfTrading(float64(price), isBuySide, true, minPrice, maxPrice)
		lor.LiquidityOrder.LiquidityImpliedSize = uint64(math.Ceil(liquidityObligation * lor.LiquidityOrder.NormalisedFraction / prob))
	}
}

// TODO: not sure if should get a price in this way, if do we need to assure that the order price has already been updated.
func (e Engine) getPrice(lor types.LiquidityOrderReference) (uint64, error) {
	o, err := e.op.GetOrderByID(lor.OrderID)
	if err != nil {
		return 0, err
	}
	return o.Price, nil
}

// TODO: This should be moved elsewhere an only called when Proportion in any of the LiquidityOrders changes
func updateNormalisedFractions(lors []*types.LiquidityOrderReference) {

	fractions := make([]float64, len(lors))
	var sum uint32 = 0
	for _, lor := range lors {
		sum += lor.LiquidityOrder.Proportion
	}
	fpSum := float64(sum)

	for _, lor := range lors {
		lor.LiquidityOrder.NormalisedFraction = float64(lor.LiquidityOrder.Proportion) / fpSum
	}
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
