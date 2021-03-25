package products

import (
	"context"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/oracles"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

var (
	// ErrOracleSpecAndBindingAreRequired is returned when the definition of the
	// oracle spec or its binding is missing from the future definition.
	ErrOracleSpecAndBindingAreRequired = errors.New("an oracle spec and an oracle spec binding are required")
)

// Future represent a Future as describe by the market framework
type Future struct {
	log             *logging.Logger
	SettlementAsset string
	QuoteName       string
	Maturity        time.Time
	oracle          oracle
}

type oracle struct {
	spec           *oracles.OracleSpec
	subscriptionID oracles.SubscriptionID
	binding        oracleBinding
	data           oracleData
}

type oracleData struct {
	updated         bool
	settlementPrice int64
}

func (d *oracleData) SettlementPrice() (int64, error) {
	if !d.updated {
		return 0, errors.New("settlement price is not set")
	}
	return d.settlementPrice, nil
}

type oracleBinding struct {
	settlementPriceProperty string
}

// Settle a position against the future
func (f *Future) Settle(entryPrice uint64, netPosition int64) (*types.FinancialAmount, error) {
	settlementPrice, err := f.oracle.data.SettlementPrice()
	if err != nil {
		return nil, err
	}

	// Make sure net position is positive
	if netPosition < 0 {
		netPosition = 0 - netPosition
	}

	sPrice := uint64(settlementPrice)
	var amount uint64
	if sPrice > entryPrice {
		amount = (sPrice - entryPrice) * uint64(netPosition)
	} else {
		amount = (entryPrice - sPrice) * uint64(netPosition)
	}

	return &types.FinancialAmount{
		Asset:  f.SettlementAsset,
		Amount: amount,
	}, nil
}

// Value - returns the nominal value of a unit given a current mark price
func (f *Future) Value(markPrice uint64) (uint64, error) {
	return markPrice, nil
}

// GetAsset return the asset used by the future
func (f *Future) GetAsset() string {
	return f.SettlementAsset
}

func (f *Future) updateSettlementPrice(ctx context.Context, data oracles.OracleData) error {
	if f.log.GetLevel() == logging.DebugLevel {
		f.log.Debug("new oracle data received", data.Debug()...)
	}

	settlementPrice, err := data.GetInteger(f.oracle.binding.settlementPriceProperty)
	if err != nil {
		f.log.Error(
			"could not parse the property acting as settlement price",
			logging.Error(err),
		)
		return err
	}

	f.oracle.data.settlementPrice = settlementPrice
	f.oracle.data.updated = true

	if f.log.GetLevel() == logging.DebugLevel {
		f.log.Debug(
			"future settlement price updated",
			logging.Int64("settlementPrice", settlementPrice),
		)
	}

	return nil
}

func newFuture(ctx context.Context, log *logging.Logger, f *types.Future, oe OracleEngine) (*Future, error) {
	maturity, err := time.Parse(time.RFC3339, f.Maturity)
	if err != nil {
		return nil, errors.Wrap(err, "invalid maturity time format")
	}

	if f.OracleSpec == nil || f.OracleSpecBinding == nil {
		return nil, ErrOracleSpecAndBindingAreRequired
	}
	oracleSpec, err := oracles.NewOracleSpec(*f.OracleSpec)
	if err != nil {
		return nil, err
	}

	oracleBinding, err := newOracleBinding(f)
	if err != nil {
		return nil, err
	}

	if !oracleSpec.CanBindProperty(oracleBinding.settlementPriceProperty) {
		return nil, errors.New("bound settlement price property is not filtered by oracle spec")
	}

	future := &Future{
		log:             log,
		SettlementAsset: f.SettlementAsset,
		QuoteName:       f.QuoteName,
		Maturity:        maturity,
		oracle: oracle{
			spec:    oracleSpec,
			binding: oracleBinding,
		},
	}

	future.oracle.subscriptionID = oe.Subscribe(ctx, *oracleSpec, future.updateSettlementPrice)

	if log.GetLevel() == logging.DebugLevel {
		log.Debug(
			"future subscribed to oracle engine",
			logging.Uint64("subscription ID", uint64(future.oracle.subscriptionID)),
		)
	}

	return future, nil
}

func newOracleBinding(f *types.Future) (oracleBinding, error) {
	settlementPriceProperty := strings.TrimSpace(f.OracleSpecBinding.SettlementPriceProperty)
	if len(settlementPriceProperty) == 0 {
		return oracleBinding{}, errors.New("binding for settlement price cannot be blank")
	}

	return oracleBinding{
		settlementPriceProperty: settlementPriceProperty,
	}, nil
}
