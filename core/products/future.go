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

	"code.vegaprotocol.io/vega/core/datasource"
	dscommon "code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/datasource/spec"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
	"github.com/pkg/errors"
)

var (
	// ErrDataSourceSpecAndBindingAreRequired is returned when the definition of the
	// data source spec or its binding is missing from the future definition.
	ErrDataSourceSpecAndBindingAreRequired = errors.New("a data source spec and spec binding are required")

	// ErrDataSourceSettlementDataNotSet is returned when the data source has not set the settlement data.
	ErrDataSourceSettlementDataNotSet = errors.New("settlement data is not set")

	// ErrSettlementDataDecimalsNotSupportedByAsset is returned when the decimal data decimal places
	// are more than the asset decimals.
	ErrSettlementDataDecimalsNotSupportedByAsset = errors.New("settlement data decimals not suported by market asset")
)

// Future represent a Future as describe by the market framework.
type Future struct {
	log                        *logging.Logger
	SettlementAsset            string
	QuoteName                  string
	oracle                     oracle
	tradingTerminationListener func(context.Context, bool)
	settlementDataListener     func(context.Context, *num.Numeric)
	assetDP                    uint32
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
	settlementDataSubscriptionID     spec.SubscriptionID
	settlementScheduleSubscriptionID spec.SubscriptionID
	tradingTerminatedSubscriptionID  spec.SubscriptionID
	unsubscribe                      spec.Unsubscriber
	unsubscribeSchedule              spec.Unsubscriber
	binding                          oracleBinding
	data                             oracleData
}

type oracleData struct {
	settlData         *num.Numeric
	tradingTerminated bool
}

// SettlementData returns oracle data settlement data scaled as Uint.
func (o *oracleData) SettlementData(op, ap uint32) (*num.Uint, error) {
	if o.settlData.Decimal() == nil && o.settlData.Uint() == nil {
		return nil, ErrDataSourceSettlementDataNotSet
	}

	if !o.settlData.SupportDecimalPlaces(int64(ap)) {
		return nil, ErrSettlementDataDecimalsNotSupportedByAsset
	}

	// scale to given target decimals by multiplying by 10^(targetDP - oracleDP)
	// if targetDP > oracleDP - this scales up the decimals of settlement data
	// if targetDP < oracleDP - this scaled down the decimals of settlement data and can lead to loss of accuracy
	// if there're equal - no scaling happens
	return o.settlData.ScaleTo(int64(op), int64(ap))
}

// IsTradingTerminated returns true when oracle has signalled termination of trading.
func (o *oracleData) IsTradingTerminated() bool {
	return o.tradingTerminated
}

type oracleBinding struct {
	settlementDataProperty     string
	settlementDataPropertyType datapb.PropertyKey_Type
	settlementDataDecimals     uint64

	settlementScheduleProperty     string
	settlementSchedulePropertyType datapb.PropertyKey_Type

	tradingTerminationProperty string
}

func (f *Future) SubmitDataPoint(_ context.Context, _ *num.Uint, _ int64) error {
	return nil
}

func (f *Future) OnLeaveOpeningAuction(_ context.Context, _ int64) {
}

func (f *Future) GetMarginIncrease(_ int64) *num.Uint {
	return num.UintZero()
}

func (f *Future) NotifyOnSettlementData(listener func(context.Context, *num.Numeric)) {
	f.settlementDataListener = listener
}

func (f *Future) NotifyOnTradingTerminated(listener func(context.Context, bool)) {
	f.tradingTerminationListener = listener
}

func (f *Future) RestoreSettlementData(settleData *num.Numeric) {
	f.oracle.data.settlData = settleData
}

func (f *Future) ScaleSettlementDataToDecimalPlaces(price *num.Numeric, dp uint32) (*num.Uint, error) {
	if !price.SupportDecimalPlaces(int64(dp)) {
		return nil, ErrSettlementDataDecimalsNotSupportedByAsset
	}

	settlDataDecimals := int64(f.oracle.binding.settlementDataDecimals)
	return price.ScaleTo(settlDataDecimals, int64(dp))
}

// Settle a position against the future.
func (f *Future) Settle(entryPriceInAsset, settlementData *num.Uint, netFractionalPosition num.Decimal) (amt *types.FinancialAmount, neg bool, rounding num.Decimal, err error) {
	amount, neg := settlementData.Delta(settlementData, entryPriceInAsset)
	// Make sure net position is positive
	if netFractionalPosition.IsNegative() {
		netFractionalPosition = netFractionalPosition.Neg()
		neg = !neg
	}

	if f.log.IsDebug() {
		f.log.Debug("settlement",
			logging.String("entry-price-in-asset", entryPriceInAsset.String()),
			logging.String("settlement-data-in-asset", settlementData.String()),
			logging.String("net-fractional-position", netFractionalPosition.String()),
			logging.String("amount-in-decimal", netFractionalPosition.Mul(amount.ToDecimal()).String()),
			logging.String("amount-in-uint", amount.String()),
		)
	}
	a, rem := num.UintFromDecimalWithFraction(netFractionalPosition.Mul(amount.ToDecimal()))

	return &types.FinancialAmount{
		Asset:  f.SettlementAsset,
		Amount: a,
	}, neg, rem, nil
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

func (f *Future) updateTradingTerminated(ctx context.Context, data dscommon.Data) error {
	if f.log.GetLevel() == logging.DebugLevel {
		f.log.Debug("new oracle data received", data.Debug()...)
	}

	tradingTerminated, err := data.GetBoolean(f.oracle.binding.tradingTerminationProperty)

	return f.setTradingTerminated(ctx, tradingTerminated, err)
}

func (f *Future) updateTradingTerminatedByTimestamp(ctx context.Context, data dscommon.Data) error {
	if f.log.GetLevel() == logging.DebugLevel {
		f.log.Debug("new oracle data received", data.Debug()...)
	}

	var tradingTerminated bool
	var err error

	if _, err = data.GetTimestamp(spec.BuiltinTimestamp); err == nil {
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

func (f *Future) updateSettlementData(ctx context.Context, data dscommon.Data) error {
	if f.log.GetLevel() == logging.DebugLevel {
		f.log.Debug("new oracle data received", data.Debug()...)
	}

	odata := &oracleData{
		settlData: &num.Numeric{},
	}
	switch f.oracle.binding.settlementDataPropertyType {
	case datapb.PropertyKey_TYPE_DECIMAL:
		settlDataAsDecimal, err := data.GetDecimal(f.oracle.binding.settlementDataProperty)
		if err != nil {
			f.log.Error(
				"could not parse decimal type property acting as settlement data",
				logging.Error(err),
			)
			return err
		}

		odata.settlData.SetDecimal(&settlDataAsDecimal)

	default:
		settlDataAsUint, err := data.GetUint(f.oracle.binding.settlementDataProperty)
		if err != nil {
			f.log.Error(
				"could not parse integer type property acting as settlement data",
				logging.Error(err),
			)
			return err
		}

		odata.settlData.SetUint(settlDataAsUint)
	}

	f.oracle.data.settlData = odata.settlData
	if f.settlementDataListener != nil {
		f.settlementDataListener(ctx, odata.settlData)
	}

	if f.log.GetLevel() == logging.DebugLevel {
		f.log.Debug(
			"future settlement data updated",
			logging.String("settlementData", f.oracle.data.settlData.String()),
		)
	}

	return nil
}

func (f *Future) Serialize() *snapshotpb.Product {
	return &snapshotpb.Product{}
}

func NewFuture(ctx context.Context, log *logging.Logger, f *types.Future, oe OracleEngine, assetDP uint32) (*Future, error) {
	if f.DataSourceSpecForSettlementData == nil || f.DataSourceSpecForTradingTermination == nil || f.DataSourceSpecBinding == nil {
		return nil, ErrDataSourceSpecAndBindingAreRequired
	}

	oracleBinding, err := newOracleBinding(f)
	if err != nil {
		return nil, err
	}

	dSrcSpec := f.DataSourceSpecForSettlementData.GetDefinition()

	for _, f := range dSrcSpec.GetFilters() {
		// Oracle specs with more than one unique filter names are not allowed to exists, so we do not have to make that check here.
		// We are good to only check if the type is `PropertyKey_TYPE_DECIMAL` or `PropertyKey_TYPE_INTEGER`, because we take decimals
		// into consideration only in those cases.
		if f.Key.Type == datapb.PropertyKey_TYPE_INTEGER && f.Key.NumberDecimalPlaces != nil {
			oracleBinding.settlementDataPropertyType = f.Key.Type
			oracleBinding.settlementDataDecimals = *f.Key.NumberDecimalPlaces
			break
		}
	}

	future := &Future{
		log:             log,
		SettlementAsset: f.SettlementAsset,
		QuoteName:       f.QuoteName,
		oracle: oracle{
			binding: oracleBinding,
		},
		assetDP: assetDP,
	}

	// Oracle spec for settlement data.
	oracleSpecForSettlementData, err := spec.New(*datasource.SpecFromDefinition(*f.DataSourceSpecForSettlementData.Data))
	if err != nil {
		return nil, err
	}

	switch oracleBinding.settlementDataPropertyType {
	case datapb.PropertyKey_TYPE_INTEGER:
		err := oracleSpecForSettlementData.EnsureBoundableProperty(oracleBinding.settlementDataProperty, datapb.PropertyKey_TYPE_INTEGER)
		if err != nil {
			return nil, fmt.Errorf("invalid oracle spec binding for settlement data: %w", err)
		}
	case datapb.PropertyKey_TYPE_DECIMAL:
		err := oracleSpecForSettlementData.EnsureBoundableProperty(oracleBinding.settlementDataProperty, datapb.PropertyKey_TYPE_DECIMAL)
		if err != nil {
			return nil, fmt.Errorf("invalid oracle spec binding for settlement data: %w", err)
		}
	}

	// Subscribe registers a callback for a given OracleSpec that is called when an
	// OracleData matches the spec.
	future.oracle.settlementDataSubscriptionID, future.oracle.unsubscribe, err = oe.Subscribe(ctx, *oracleSpecForSettlementData, future.updateSettlementData)
	if err != nil {
		return nil, fmt.Errorf("could not subscribe to oracle engine for settlement data: %w", err)
	}

	if log.IsDebug() {
		log.Debug("future subscribed to oracle engine for settlement data",
			logging.Uint64("subscription ID", uint64(future.oracle.settlementDataSubscriptionID)),
		)
	}

	// Oracle spec for trading termination.
	oracleSpecForTerminatedMarket, err := spec.New(*datasource.SpecFromDefinition(*f.DataSourceSpecForTradingTermination.Data))
	if err != nil {
		return nil, err
	}

	var tradingTerminationPropType datapb.PropertyKey_Type
	var tradingTerminationCb spec.OnMatchedData
	if oracleBinding.tradingTerminationProperty == spec.BuiltinTimestamp {
		tradingTerminationPropType = datapb.PropertyKey_TYPE_TIMESTAMP
		tradingTerminationCb = future.updateTradingTerminatedByTimestamp
	} else {
		tradingTerminationPropType = datapb.PropertyKey_TYPE_BOOLEAN
		tradingTerminationCb = future.updateTradingTerminated
	}

	if err = oracleSpecForTerminatedMarket.EnsureBoundableProperty(
		oracleBinding.tradingTerminationProperty,
		tradingTerminationPropType,
	); err != nil {
		return nil, fmt.Errorf("invalid oracle spec binding for trading termination: %w", err)
	}

	future.oracle.tradingTerminatedSubscriptionID, _, err = oe.Subscribe(ctx, *oracleSpecForTerminatedMarket, tradingTerminationCb)
	if err != nil {
		return nil, fmt.Errorf("could not subscribe to oracle engine for trading termination: %w", err)
	}

	if log.IsDebug() {
		log.Debug("future subscribed to oracle engine for market termination event",
			logging.Uint64("subscription ID", uint64(future.oracle.tradingTerminatedSubscriptionID)),
		)
	}

	return future, nil
}

func newOracleBinding(f *types.Future) (oracleBinding, error) {
	settlementDataProperty := strings.TrimSpace(f.DataSourceSpecBinding.SettlementDataProperty)
	if len(settlementDataProperty) == 0 {
		return oracleBinding{}, errors.New("binding for settlement data cannot be blank")
	}
	tradingTerminationProperty := strings.TrimSpace(f.DataSourceSpecBinding.TradingTerminationProperty)
	if len(tradingTerminationProperty) == 0 {
		return oracleBinding{}, errors.New("binding for trading termination market cannot be blank")
	}

	return oracleBinding{
		settlementDataProperty:     settlementDataProperty,
		tradingTerminationProperty: tradingTerminationProperty,
	}, nil
}
