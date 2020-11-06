package supplied

import (
	"errors"
	"math"

	types "code.vegaprotocol.io/vega/proto"
)

var (
	// ErrNilLiquidityProvisionProvider signals that nil was supplied in place of LiquidityProvisionProvider
	ErrNilLiquidityProvisionProvider = errors.New("nil LiquidityProvisionProvider")
	// ErrNilOrderProvider signals that nil was supplied in place of OrderProvider
	ErrNilOrderProvider = errors.New("nil OrderProvider")
	// ErrNilRiskModel signals that nil was supplied in place of RiskModel
	ErrNilRiskModel = errors.New("nil RiskModel")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/liquidity_provision_provider_mock.go -package mocks code.vegaprotocol.io/vega/liquidity/supplied LiquidityProvisionProvider
// LiquidityProvisionProvider allows getting all the liquidity provisions per specied market ID
type LiquidityProvisionProvider interface {
	GetLiquidityProvisions(market string) ([]types.LiquidityProvision, error)
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
	mID string
	lpp LiquidityProvisionProvider
	op  OrderProvider
	rm  RiskModel

	//TODO: Move buys, sells here to aid memory usage
}

// NewEngine returns a reference to a new supplied liquidity calculation engine if all arguments get supplied (with non-nil values) and an error otherwise
func NewEngine(marketID string, lpProvider LiquidityProvisionProvider, orderProvider OrderProvider, riskModel RiskModel) (*Engine, error) {
	if lpProvider == nil {
		return nil, ErrNilLiquidityProvisionProvider
	}
	if orderProvider == nil {
		return nil, ErrNilOrderProvider
	}
	if riskModel == nil {
		return nil, ErrNilRiskModel
	}

	return &Engine{
		mID: marketID,
		lpp: lpProvider,
		op:  orderProvider,
		rm:  riskModel,
	}, nil
}

// GetSuppliedLiquidity returns the current supplied liquidity per market specified in the constructor
func (e Engine) GetSuppliedLiquidity() (float64, error) {
	buys, sells, err := e.getLiquidityProvisionOrders()
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

func (e Engine) getLiquidityProvisionOrders() (map[uint64]uint64, map[uint64]uint64, error) {
	lps, err := e.lpp.GetLiquidityProvisions(e.mID)
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

// TODO: Do we need a liquidity engine that liqudity service will reference? Then we could pass reference to that engine to market and use it here.
