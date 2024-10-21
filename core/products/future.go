// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package products

import (
	"context"

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
	oracle                     terminatingOracle
	tradingTerminationListener func(context.Context, bool)
	settlementDataListener     func(context.Context, *num.Numeric)
	assetDP                    uint32
}

func (_ Future) GetCurrentPeriod() uint64 { return 0 }

func (f *Future) UnsubscribeTradingTerminated(ctx context.Context) {
	f.log.Info("unsubscribed trading terminated for", logging.String("quote-name", f.QuoteName))
	f.oracle.unsubTerm(ctx)
}

func (f *Future) UnsubscribeSettlementData(ctx context.Context) {
	f.log.Info("unsubscribed trading settlement data for", logging.String("quote-name", f.QuoteName))
	f.oracle.unsubSettle(ctx)
}

func (f *Future) Unsubscribe(ctx context.Context) {
	f.UnsubscribeTradingTerminated(ctx)
	f.UnsubscribeSettlementData(ctx)
}

func (f *Future) SubmitDataPoint(_ context.Context, _ *num.Uint, _ int64) error {
	return nil
}

func (f *Future) UpdateAuctionState(_ context.Context, _ bool) {
}

func (f *Future) GetMarginIncrease(_ int64) num.Decimal {
	return num.DecimalZero()
}

func (f *Future) NotifyOnDataSourcePropagation(listener func(context.Context, *num.Uint)) {
	f.log.Panic("not implemented")
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

	settlDataDecimals := int64(f.oracle.binding.settlementDecimals)
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

	tradingTerminated, err := data.GetBoolean(f.oracle.binding.terminationProperty)

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
	switch f.oracle.binding.settlementType {
	case datapb.PropertyKey_TYPE_DECIMAL:
		settlDataAsDecimal, err := data.GetDecimal(f.oracle.binding.settlementProperty)
		if err != nil {
			f.log.Error(
				"could not parse decimal type property acting as settlement data",
				logging.Error(err),
			)
			return err
		}

		odata.settlData.SetDecimal(&settlDataAsDecimal)

	default:
		settlDataAsUint, err := data.GetUint(f.oracle.binding.settlementProperty)
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

func (f *Future) Update(ctx context.Context, pp interface{}, oe OracleEngine) error {
	ff, ok := pp.(*types.InstrumentFuture)
	if !ok {
		f.log.Panic("attempting to update a future into something else")
	}

	cfg := ff.Future

	// unsubscribe the old data sources
	f.oracle.unsubSettle(ctx)
	f.oracle.unsubTerm(ctx)

	oracle, err := newFutureOracle(cfg)
	if err != nil {
		return err
	}

	// subscribe to new
	// Oracle spec for settlement data.
	osForSettle, err := spec.New(*datasource.SpecFromDefinition(*cfg.DataSourceSpecForSettlementData.Data))
	if err != nil {
		return err
	}
	osForTerm, err := spec.New(*datasource.SpecFromDefinition(*cfg.DataSourceSpecForTradingTermination.Data))
	if err != nil {
		return err
	}
	tradingTerminationCb := f.updateTradingTerminated
	if oracle.binding.terminationType == datapb.PropertyKey_TYPE_TIMESTAMP {
		tradingTerminationCb = f.updateTradingTerminatedByTimestamp
	}
	if err := oracle.bindAll(ctx, oe, osForSettle, osForTerm, f.updateSettlementData, tradingTerminationCb); err != nil {
		return err
	}

	f.oracle = oracle
	return nil
}

func (f *Future) GetData(t int64) *types.ProductData {
	return nil
}

func NewFuture(ctx context.Context, log *logging.Logger, f *types.Future, oe OracleEngine, assetDP uint32) (*Future, error) {
	if f.DataSourceSpecForSettlementData == nil || f.DataSourceSpecForTradingTermination == nil || f.DataSourceSpecBinding == nil {
		return nil, ErrDataSourceSpecAndBindingAreRequired
	}

	oracle, err := newFutureOracle(f)
	if err != nil {
		return nil, err
	}

	future := &Future{
		log:             log,
		SettlementAsset: f.SettlementAsset,
		QuoteName:       f.QuoteName,
		assetDP:         assetDP,
	}

	// Oracle spec for settlement data.
	osForSettle, err := spec.New(*datasource.SpecFromDefinition(*f.DataSourceSpecForSettlementData.Data))
	if err != nil {
		return nil, err
	}
	osForTerm, err := spec.New(*datasource.SpecFromDefinition(*f.DataSourceSpecForTradingTermination.Data))
	if err != nil {
		return nil, err
	}
	tradingTerminationCb := future.updateTradingTerminated
	if oracle.binding.terminationType == datapb.PropertyKey_TYPE_TIMESTAMP {
		tradingTerminationCb = future.updateTradingTerminatedByTimestamp
	}
	if err := oracle.bindAll(ctx, oe, osForSettle, osForTerm, future.updateSettlementData, tradingTerminationCb); err != nil {
		return nil, err
	}
	future.oracle = oracle // ensure the oracle on future is not an old copy

	if log.IsDebug() {
		log.Debug("future subscribed to oracle engine for settlement data",
			logging.Uint64("subscription ID", uint64(future.oracle.settlementSubscriptionID)),
		)
		log.Debug("future subscribed to oracle engine for market termination event",
			logging.Uint64("subscription ID", uint64(future.oracle.terminationSubscriptionID)),
		)
	}

	return future, nil
}
