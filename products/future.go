package products

import (
	"context"
	"fmt"
	"strings"

	oraclespb "code.vegaprotocol.io/protos/vega/oracles/v1"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/oracles"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/pkg/errors"
)

var (
	// ErrOracleSpecAndBindingAreRequired is returned when the definition of the
	// oracle spec or its binding is missing from the future definition.
	ErrOracleSpecAndBindingAreRequired = errors.New("an oracle spec and an oracle spec binding are required")

	// ErrOracleSettlementPriceNotSet is returned when the oracle has not set the settlement price.
	ErrOracleSettlementPriceNotSet = errors.New("settlement price is not set")
)

// Future represent a Future as describe by the market framework.
type Future struct {
	log                        *logging.Logger
	SettlementAsset            string
	QuoteName                  string
	oracle                     oracle
	tradingTerminationListener func(context.Context, bool)
	settlementPriceListener    func(context.Context, *num.Uint)
}

type oracle struct {
	settlementPriceSubscriptionID   oracles.SubscriptionID
	tradingTerminatedSubscriptionID oracles.SubscriptionID
	binding                         oracleBinding
	data                            oracleData
}

type oracleData struct {
	settlementPrice   *num.Uint
	tradingTerminated bool
}

func (d *oracleData) SettlementPrice() (*num.Uint, error) {
	if d.settlementPrice == nil {
		return nil, ErrOracleSettlementPriceNotSet
	}
	return d.settlementPrice.Clone(), nil
}

// IsTradingTerminated returns true when oracle has signalled termination of trading.
func (d *oracleData) IsTradingTerminated() bool {
	return d.tradingTerminated
}

type oracleBinding struct {
	settlementPriceProperty    string
	tradingTerminationProperty string
}

func (f *Future) NotifyOnSettlementPrice(listener func(context.Context, *num.Uint)) {
	f.settlementPriceListener = listener
}

func (f *Future) NotifyOnTradingTerminated(listener func(context.Context, bool)) {
	f.tradingTerminationListener = listener
}

func (f *Future) SettlementPrice() (*num.Uint, error) {
	return f.oracle.data.SettlementPrice()
}

// Settle a position against the future.
func (f *Future) Settle(entryPrice *num.Uint, netFractionalPosition num.Decimal) (amt *types.FinancialAmount, neg bool, err error) {
	settlementPrice, err := f.oracle.data.SettlementPrice()
	if err != nil {
		return nil, false, err
	}

	amount, neg := settlementPrice.Delta(settlementPrice, entryPrice)
	// Make sure net position is positive
	if netFractionalPosition.IsNegative() {
		netFractionalPosition = netFractionalPosition.Neg()
		neg = !neg
	}

	amount, _ = num.UintFromDecimal(netFractionalPosition.Mul(amount.ToDecimal()))

	return &types.FinancialAmount{
		Asset:  f.SettlementAsset,
		Amount: amount,
	}, neg, nil
}

// Value - returns the nominal value of a unit given a current mark price.
func (f *Future) Value(markPrice *num.Uint) (*num.Uint, error) {
	return markPrice.Clone(), nil
}

// IsTradingTerminated - returns true when the oracle has signalled terminated market.
func (f *Future) IsTradingTerminated() bool {
	return f.oracle.data.IsTradingTerminated()
}

// GetAsset return the asset used by the future.
func (f *Future) GetAsset() string {
	return f.SettlementAsset
}

func (f *Future) updateTradingTerminated(ctx context.Context, data oracles.OracleData) error {
	if f.log.GetLevel() == logging.DebugLevel {
		f.log.Debug("new oracle data received", data.Debug()...)
	}

	tradingTerminated, err := data.GetBoolean(f.oracle.binding.tradingTerminationProperty)

	return f.setTradingTerminated(ctx, tradingTerminated, err)
}

func (f *Future) updateTradingTerminatedByTimestamp(ctx context.Context, data oracles.OracleData) error {
	if f.log.GetLevel() == logging.DebugLevel {
		f.log.Debug("new oracle data received", data.Debug()...)
	}

	var tradingTerminated bool
	var err error

	if _, err = data.GetTimestamp(oracles.BuiltinOracleTimestamp); err == nil {
		// we have received a trading termination timestamp from the internal vega time oracle
		tradingTerminated = true
	}

	return f.setTradingTerminated(ctx, tradingTerminated, err)
}

func (f *Future) setTradingTerminated(ctx context.Context, tradingTerminated bool, dataErr error) error {
	if dataErr != nil {
		f.log.Error(
			"could not parse the property acting as trading Terminated",
			logging.Error(dataErr),
		)
		return dataErr
	}

	f.oracle.data.tradingTerminated = tradingTerminated
	if f.tradingTerminationListener != nil {
		f.tradingTerminationListener(ctx, tradingTerminated)
	}
	return nil
}

func (f *Future) updateSettlementPrice(ctx context.Context, data oracles.OracleData) error {
	if f.log.GetLevel() == logging.DebugLevel {
		f.log.Debug("new oracle data received", data.Debug()...)
	}

	settlementPrice, err := data.GetUint(f.oracle.binding.settlementPriceProperty)
	if err != nil {
		f.log.Error(
			"could not parse the property acting as settlement price",
			logging.Error(err),
		)
		return err
	}

	f.oracle.data.settlementPrice = settlementPrice
	if f.settlementPriceListener != nil {
		f.settlementPriceListener(ctx, settlementPrice)
	}

	if f.log.GetLevel() == logging.DebugLevel {
		f.log.Debug(
			"future settlement price updated",
			logging.BigUint("settlementPrice", settlementPrice),
		)
	}

	return nil
}

func newFuture(ctx context.Context, log *logging.Logger, f *types.Future, oe OracleEngine) (*Future, error) {
	if f.OracleSpecForSettlementPrice == nil || f.OracleSpecForTradingTermination == nil || f.OracleSpecBinding == nil {
		return nil, ErrOracleSpecAndBindingAreRequired
	}

	oracleBinding, err := newOracleBinding(f)
	if err != nil {
		return nil, err
	}

	future := &Future{
		log:             log,
		SettlementAsset: f.SettlementAsset,
		QuoteName:       f.QuoteName,
		oracle: oracle{
			binding: oracleBinding,
		},
	}

	// Oracle spec for settlement price.
	oracleSpecForSettlementPrice, err := oracles.NewOracleSpec(*f.OracleSpecForSettlementPrice)
	if err != nil {
		return nil, err
	}

	if err := oracleSpecForSettlementPrice.EnsureBoundableProperty(
		oracleBinding.settlementPriceProperty,
		oraclespb.PropertyKey_TYPE_INTEGER,
	); err != nil {
		return nil, fmt.Errorf("invalid oracle spec binding for settlement price: %w", err)
	}

	future.oracle.settlementPriceSubscriptionID = oe.Subscribe(ctx, *oracleSpecForSettlementPrice, future.updateSettlementPrice)

	if log.IsDebug() {
		log.Debug("future subscribed to oracle engine for settlement price",
			logging.Uint64("subscription ID", uint64(future.oracle.settlementPriceSubscriptionID)),
		)
	}

	// Oracle spec for trading termination.
	oracleSpecForTerminatedMarket, err := oracles.NewOracleSpec(*f.OracleSpecForTradingTermination)
	if err != nil {
		return nil, err
	}

	var tradingTerminationPropType oraclespb.PropertyKey_Type
	var tradingTerminationCb oracles.OnMatchedOracleData
	if oracleBinding.tradingTerminationProperty == oracles.BuiltinOracleTimestamp {
		tradingTerminationPropType = oraclespb.PropertyKey_TYPE_TIMESTAMP
		tradingTerminationCb = future.updateTradingTerminatedByTimestamp
	} else {
		tradingTerminationPropType = oraclespb.PropertyKey_TYPE_BOOLEAN
		tradingTerminationCb = future.updateTradingTerminated
	}

	if err = oracleSpecForTerminatedMarket.EnsureBoundableProperty(
		oracleBinding.tradingTerminationProperty,
		tradingTerminationPropType,
	); err != nil {
		return nil, fmt.Errorf("invalid oracle spec binding for trading termination: %w", err)
	}

	future.oracle.tradingTerminatedSubscriptionID = oe.Subscribe(ctx, *oracleSpecForTerminatedMarket, tradingTerminationCb)

	if log.IsDebug() {
		log.Debug("future subscribed to oracle engine for market termination event",
			logging.Uint64("subscription ID", uint64(future.oracle.tradingTerminatedSubscriptionID)),
		)
	}

	return future, nil
}

func newOracleBinding(f *types.Future) (oracleBinding, error) {
	settlementPriceProperty := strings.TrimSpace(f.OracleSpecBinding.SettlementPriceProperty)
	if len(settlementPriceProperty) == 0 {
		return oracleBinding{}, errors.New("binding for settlement price cannot be blank")
	}
	tradingTerminationProperty := strings.TrimSpace(f.OracleSpecBinding.TradingTerminationProperty)
	if len(tradingTerminationProperty) == 0 {
		return oracleBinding{}, errors.New("binding for trading termination market cannot be blank")
	}

	return oracleBinding{
		settlementPriceProperty:    settlementPriceProperty,
		tradingTerminationProperty: tradingTerminationProperty,
	}, nil
}
