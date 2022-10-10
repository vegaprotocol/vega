// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package products

import (
	"context"
	"fmt"
	"strings"

	"code.vegaprotocol.io/vega/core/oracles"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	oraclespb "code.vegaprotocol.io/vega/protos/vega/oracles/v1"
	"github.com/pkg/errors"
)

var (
	// ErrOracleSpecAndBindingAreRequired is returned when the definition of the
	// oracle spec or its binding is missing from the future definition.
	ErrOracleSpecAndBindingAreRequired = errors.New("an oracle spec and an oracle spec binding are required")

	// ErrOracleSettlementDataeNotSet is returned when the oracle has not set the settlement data.
	ErrOracleSettlementDataNotSet = errors.New("settlement data is not set")
)

// Future represent a Future as describe by the market framework.
type Future struct {
	log                        *logging.Logger
	SettlementAsset            string
	QuoteName                  string
	oracle                     oracle
	tradingTerminationListener func(context.Context, bool)
	settlementDataListener     func(context.Context, *num.Uint)
}

func (f *Future) UnsubscribeTradingTerminated(ctx context.Context) {
	f.log.Info("unsubscribed trading terminated for", logging.String("quote-name", f.QuoteName))
	f.oracle.unsubscribe(ctx, f.oracle.tradingTerminatedSubscriptionID)
}

func (f *Future) UnsubscribeSettlementData(ctx context.Context) {
	f.log.Info("unsubscribed trading settlement data for", logging.String("quote-name", f.QuoteName))
	f.oracle.unsubscribe(ctx, f.oracle.settlementDataSubscriptionID)
}

func (f *Future) Unsubscribe(ctx context.Context) {
	f.UnsubscribeTradingTerminated(ctx)
	f.UnsubscribeSettlementData(ctx)
}

type oracle struct {
	settlementDataSubscriptionID    oracles.SubscriptionID
	tradingTerminatedSubscriptionID oracles.SubscriptionID
	unsubscribe                     oracles.Unsubscriber
	binding                         oracleBinding
	data                            oracleData
	settlementDataDecimals          uint32
}

type oracleData struct {
	settlementData    *num.Uint
	tradingTerminated bool
}

func (d *oracleData) SettlementData() (*num.Uint, error) {
	if d.settlementData == nil {
		return nil, ErrOracleSettlementDataNotSet
	}
	return d.settlementData.Clone(), nil
}

// IsTradingTerminated returns true when oracle has signalled termination of trading.
func (d *oracleData) IsTradingTerminated() bool {
	return d.tradingTerminated
}

type oracleBinding struct {
	settlementDataProperty     string
	tradingTerminationProperty string
}

func (f *Future) NotifyOnSettlementData(listener func(context.Context, *num.Uint)) {
	f.settlementDataListener = listener
}

func (f *Future) NotifyOnTradingTerminated(listener func(context.Context, bool)) {
	f.tradingTerminationListener = listener
}

func (f *Future) ScaleSettlementDataToDecimalPlaces(price *num.Uint, dp uint32) (*num.Uint, error) {
	// scale to asset decimals by multiplying by 10^(assetDP - oracleDP)
	// if assetDP > oracleDP - this scales up the decimals of settlement data
	// if assetDP < oracleDP - this scaled down the decimals of settlement data and can lead to loss of accuracy
	// if there're equal - no scaling happens
	scalingFactor := num.DecimalFromInt64(10).Pow(num.DecimalFromInt64(int64(dp) - int64(f.oracle.settlementDataDecimals)))
	r, overflow := num.UintFromDecimal(price.ToDecimal().Mul(scalingFactor))
	if overflow {
		return nil, errors.New("failed to scale settlement data, overflow occurred")
	}
	return r, nil
}

// Settle a position against the future.
func (f *Future) Settle(entryPriceInAsset *num.Uint, assetDecimals uint32, netFractionalPosition num.Decimal) (amt *types.FinancialAmount, neg bool, err error) {
	settlementData, err := f.oracle.data.SettlementData()
	if err != nil {
		return nil, false, err
	}

	settlementDataInAsset, err := f.ScaleSettlementDataToDecimalPlaces(settlementData, assetDecimals)
	if err != nil {
		return nil, false, err
	}

	amount, neg := settlementData.Delta(settlementDataInAsset, entryPriceInAsset)
	// Make sure net position is positive
	if netFractionalPosition.IsNegative() {
		netFractionalPosition = netFractionalPosition.Neg()
		neg = !neg
	}

	if f.log.IsDebug() {
		f.log.Debug("settlement",
			logging.String("entry-price-in-asset", entryPriceInAsset.String()),
			logging.String("settlement-data-in-asset", settlementDataInAsset.String()),
			logging.String("net-fractional-position", netFractionalPosition.String()),
			logging.String("amount-in-decimal", netFractionalPosition.Mul(amount.ToDecimal()).String()),
			logging.String("amount-in-uint", amount.String()),
		)
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

func (f *Future) updateSettlementData(ctx context.Context, data oracles.OracleData) error {
	if f.log.GetLevel() == logging.DebugLevel {
		f.log.Debug("new oracle data received", data.Debug()...)
	}

	settlementData, err := data.GetUint(f.oracle.binding.settlementDataProperty)
	if err != nil {
		f.log.Error(
			"could not parse the property acting as settlement data",
			logging.Error(err),
		)
		return err
	}

	f.oracle.data.settlementData = settlementData
	if f.settlementDataListener != nil {
		f.settlementDataListener(ctx, settlementData)
	}

	if f.log.GetLevel() == logging.DebugLevel {
		f.log.Debug(
			"future settlement data updated",
			logging.BigUint("settlementData", settlementData),
		)
	}

	return nil
}

func NewFuture(ctx context.Context, log *logging.Logger, f *types.Future, oe OracleEngine) (*Future, error) {
	if f.OracleSpecForSettlementData == nil || f.OracleSpecForTradingTermination == nil || f.OracleSpecBinding == nil {
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
			binding:                oracleBinding,
			settlementDataDecimals: f.SettlementDataDecimals,
		},
	}

	// Oracle spec for settlement data.
	OracleSpecForSettlementData, err := oracles.NewOracleSpec(*f.OracleSpecForSettlementData)
	if err != nil {
		return nil, err
	}

	if err := OracleSpecForSettlementData.EnsureBoundableProperty(
		oracleBinding.settlementDataProperty,
		oraclespb.PropertyKey_TYPE_INTEGER,
	); err != nil {
		return nil, fmt.Errorf("invalid oracle spec binding for settlement data: %w", err)
	}

	future.oracle.settlementDataSubscriptionID, future.oracle.unsubscribe = oe.Subscribe(ctx, *OracleSpecForSettlementData, future.updateSettlementData)

	if log.IsDebug() {
		log.Debug("future subscribed to oracle engine for settlement data",
			logging.Uint64("subscription ID", uint64(future.oracle.settlementDataSubscriptionID)),
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

	future.oracle.tradingTerminatedSubscriptionID, _ = oe.Subscribe(ctx, *oracleSpecForTerminatedMarket, tradingTerminationCb)

	if log.IsDebug() {
		log.Debug("future subscribed to oracle engine for market termination event",
			logging.Uint64("subscription ID", uint64(future.oracle.tradingTerminatedSubscriptionID)),
		)
	}

	return future, nil
}

func newOracleBinding(f *types.Future) (oracleBinding, error) {
	settlementDataProperty := strings.TrimSpace(f.OracleSpecBinding.SettlementDataProperty)
	if len(settlementDataProperty) == 0 {
		return oracleBinding{}, errors.New("binding for settlement data cannot be blank")
	}
	tradingTerminationProperty := strings.TrimSpace(f.OracleSpecBinding.TradingTerminationProperty)
	if len(tradingTerminationProperty) == 0 {
		return oracleBinding{}, errors.New("binding for trading termination market cannot be blank")
	}

	return oracleBinding{
		settlementDataProperty:     settlementDataProperty,
		tradingTerminationProperty: tradingTerminationProperty,
	}, nil
}
