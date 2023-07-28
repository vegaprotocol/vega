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
	"strconv"
	"strings"

	"code.vegaprotocol.io/vega/core/datasource"
	dscommon "code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/datasource/spec"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/logging"
	"github.com/pkg/errors"

	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

var (
	ErrDataPointAlreadyExistsAtTime = errors.New("data-point already exists at timestamp")
	ErrInitialPeriodNotStarted      = errors.New("initial settlement period not started")
)

type dataPointSource = eventspb.FundingPeriodDataPoint_Source

const (
	dataPointSourceExternal dataPointSource = eventspb.FundingPeriodDataPoint_SOURCE_EXTERNAL
	dataPointSourceInternal dataPointSource = eventspb.FundingPeriodDataPoint_SOURCE_INTERNAL
)

// A data-point that will be used to calculate periodic settlement in a perps market.
type dataPoint struct {
	// the asset price
	price *num.Uint
	// the timestamp of this data point
	t int64
}

/*type oracle struct {
	settlementDataSubscriptionID    spec.SubscriptionID
	tradingTerminatedSubscriptionID spec.SubscriptionID
	unsubscribe                     spec.Unsubscriber
	binding                         oracleBinding
	data                            oracleData
}*/

// Perpetual represents a Perpetual as describe by the market framework.
type Perpetual struct {
	p               *types.Perps
	log             *logging.Logger
	SettlementAsset string
	QuoteName       string
	// oracle                 oracle
	settlementDataListener func(context.Context, *num.Numeric)
	broker                 Broker
	oracle                 oracle

	// id should be the same as the market id
	id string
	// data-points created externally such as spot prices received from external data-sources
	external []*dataPoint
	// data-points created internally such as MTM mark prices
	internal []*dataPoint
	// enumeration of the settlement period so that we can track which points landed in each interval
	seq uint64
	// the time that this period interval started
	startedAt int64
	// asset decimal places
	assetDP uint32
}

func NewPerpetual(ctx context.Context, log *logging.Logger, p *types.Perps, oe OracleEngine, broker Broker, assetDP uint32) (*Perpetual, error) {
	// make sure we have all we need
	if p.DataSourceSpecForSettlementData == nil || p.DataSourceSpecForSettlementSchedule == nil || p.DataSourceSpecBinding == nil {
		return nil, ErrDataSourceSpecAndBindingAreRequired
	}
	oracleBinding, err := newPerpOracleBinding(p)
	if err != nil {
		return nil, err
	}

	dsSpec := p.DataSourceSpecForSettlementData.GetDefinition()

	// check decimal places for settlement data
	for _, f := range dsSpec.GetFilters() {
		if f.Key.Type == datapb.PropertyKey_TYPE_INTEGER && f.Key.NumberDecimalPlaces != nil {
			oracleBinding.settlementDataPropertyType = f.Key.Type
			oracleBinding.settlementDataDecimals = *f.Key.NumberDecimalPlaces
			break
		}
	}
	perp := &Perpetual{
		p:       p,
		log:     log,
		broker:  broker,
		assetDP: assetDP,
		oracle: oracle{
			binding: oracleBinding,
		},
	}
	oracleSpecForSettlementData, err := spec.New(*datasource.SpecFromDefinition(*p.DataSourceSpecForSettlementData.Data))
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
	perp.oracle.settlementDataSubscriptionID, perp.oracle.unsubscribe, err = oe.Subscribe(ctx, *oracleSpecForSettlementData, perp.receiveDataPoint)
	if err != nil {
		return nil, fmt.Errorf("could not subscribe to oracle engine for settlement data: %w", err)
	}

	// set up the oracle
	return perp, nil
}

func (p *Perpetual) RestoreSettlementData(settleData *num.Numeric) {
	p.log.Panic("not implemented")
}

// NotifyOnSettlementData for a perpetual this will be the funding payment being sent to the listener.
func (p *Perpetual) NotifyOnSettlementData(listener func(context.Context, *num.Numeric)) {
	p.settlementDataListener = listener
}

func (p *Perpetual) NotifyOnTradingTerminated(listener func(context.Context, bool)) {
	p.log.Panic("not expecting trading terminated with perpetual")
}

func (p *Perpetual) ScaleSettlementDataToDecimalPlaces(price *num.Numeric, dp uint32) (*num.Uint, error) {
	p.log.Panic("not implemented")
	return nil, nil
}

// Settle a position against the perpetual.
func (p *Perpetual) Settle(entryPriceInAsset, settlementData *num.Uint, netFractionalPosition num.Decimal) (amt *types.FinancialAmount, neg bool, rounding num.Decimal, err error) {
	p.log.Panic("not implemented")
	return nil, false, num.DecimalZero(), nil
}

// Value - returns the nominal value of a unit given a current mark price.
func (p *Perpetual) Value(markPrice *num.Uint) (*num.Uint, error) {
	return markPrice.Clone(), nil
}

// IsTradingTerminated - returns true when the oracle has signalled terminated market.
func (p *Perpetual) IsTradingTerminated() bool {
	return false
}

// GetAsset return the asset used by the future.
func (p *Perpetual) GetAsset() string {
	return p.SettlementAsset
}

func (p *Perpetual) UnsubscribeTradingTerminated(ctx context.Context) {
	p.log.Panic("not expecting trading terminated with perpetual")
}

func (p *Perpetual) UnsubscribeSettlementData(ctx context.Context) {
	p.log.Panic("not implemented")
}

func (p *Perpetual) OnLeaveOpeningAuction(ctx context.Context, t int64) {
	p.startedAt = t
	p.broker.Send(events.NewFundingPeriodEvent(ctx, p.id, p.seq, p.startedAt, nil, nil, nil, nil, nil))
}

// SubmitDataPoint this will add a data point produced internally by the core node.
func (p *Perpetual) SubmitDataPoint(ctx context.Context, price *num.Uint, t int64) error {
	if !p.readyForData() {
		return ErrInitialPeriodNotStarted
	}

	point := &dataPoint{price: price.Clone(), t: t}
	if err := p.addInternal(point); err != nil {
		p.log.Error("unable to add internal data-point", logging.Error(err))
		return err
	}
	p.broker.Send(events.NewFundingPeriodDataPointEvent(ctx, p.id, price.String(), t, p.seq, dataPointSourceInternal))
	return nil
}

func (p *Perpetual) receiveDataPoint(ctx context.Context, data dscommon.Data) error {
	if p.log.GetLevel() == logging.DebugLevel {
		p.log.Debug("new oracle data received", data.Debug()...)
	}

	settlDataDecimals := int64(p.oracle.binding.settlementDataDecimals)
	odata := &oracleData{
		settlData: &num.Numeric{},
	}
	switch p.oracle.binding.settlementDataPropertyType {
	case datapb.PropertyKey_TYPE_DECIMAL:
		settlDataAsDecimal, err := data.GetDecimal(p.oracle.binding.settlementDataProperty)
		if err != nil {
			p.log.Error(
				"could not parse decimal type property acting as settlement data",
				logging.Error(err),
			)
			return err
		}

		odata.settlData.SetDecimal(&settlDataAsDecimal)

	default:
		settlDataAsUint, err := data.GetUint(p.oracle.binding.settlementDataProperty)
		if err != nil {
			p.log.Error(
				"could not parse integer type property acting as settlement data",
				logging.Error(err),
			)
			return err
		}

		odata.settlData.SetUint(settlDataAsUint)
	}

	// get scaled uint
	assetPrice, err := odata.settlData.ScaleTo(settlDataDecimals, int64(p.assetDP))
	if err != nil {
		p.log.Error("Could not scale the settle data received to asset decimals",
			logging.String("settle-data", odata.settlData.String()),
			logging.Error(err),
		)
		return err
	}
	// add price point with "eth-block-time" as time
	pTime, err := strconv.ParseUint(data.MetaData["eth-block-time"], 10, 64)
	if err != nil {
		p.log.Error("Could not parse the eth block time",
			logging.String("eth-block-time", data.MetaData["eth-block-time"]),
			logging.Error(err),
		)
		return err
	}
	// now add the price
	p.addExternalDataPoint(ctx, assetPrice, int64(pTime))
	if p.log.GetLevel() == logging.DebugLevel {
		p.log.Debug(
			"perp settlement data updated",
			logging.String("settlementData", odata.settlData.String()),
		)
	}

	return nil
}

// receiveDataPoint will be hooked up as a subscriber to the oracle data for incoming settlement data from a data-source.
func (p *Perpetual) addExternalDataPoint(ctx context.Context, price *num.Uint, t int64) {
	if !p.readyForData() {
		p.log.Error("external data point for perpetual received before initial period", logging.String("id", p.id), logging.Int64("t", t))
		return
	}

	point := &dataPoint{price: price.Clone(), t: t}
	if err := p.addExternal(point); err != nil {
		p.log.Error("unable to add external data-point", logging.Error(err))
		return
	}
	p.broker.Send(events.NewFundingPeriodDataPointEvent(ctx, p.id, price.String(), t, p.seq, dataPointSourceExternal))
}

// receiveSettlementCue will be hooked up as a subscriber to the oracle data for the notification that the settlement period has ended.
func (p *Perpetual) receiveSettlementCue(ctx context.Context, t int64) {
	if !p.haveData(t) {
		// we have no points so we just start a new interval
		p.broker.Send(events.NewFundingPeriodEvent(ctx, p.id, p.seq, p.startedAt, ptr.From(t), nil, nil, nil, nil))
		p.startNewFundingPeriod(ctx, t)
		return
	}

	internalTWAP := twap(p.internal, p.startedAt, t)
	// and calculate the same using the external oracle data-points over the same period
	externalTWAP := twap(p.external, p.startedAt, t)
	// do the calculation
	fundingPayment, fundingRate := p.calculateFundingPayment(internalTWAP, externalTWAP)

	// send it away!
	fp := &num.Numeric{}
	p.settlementDataListener(ctx, fp.SetInt(fundingPayment))

	// now restart the interval
	p.broker.Send(events.NewFundingPeriodEvent(ctx, p.id, p.seq, p.startedAt, ptr.From(t), ptr.From(fundingPayment.String()), ptr.From(fundingRate.String()), ptr.From(internalTWAP.String()), ptr.From(externalTWAP.String())))
	p.startNewFundingPeriod(ctx, t)
}

// restarts the funcing period at time st.
func (p *Perpetual) startNewFundingPeriod(ctx context.Context, endAt int64) {
	// base the TWAP estimate of this period on the duration of last period
	duration := endAt - p.startedAt
	// increment seq and set start to the time the previous ended
	p.seq += 1
	p.startedAt = endAt
	p.log.Info("new settlement period",
		logging.MarketID(p.id),
		logging.Int64("t", endAt),
	)

	carryOver := func(points []*dataPoint) []*dataPoint {
		carry := []*dataPoint{}
		for i := len(points) - 1; i >= 0; i-- {
			carry = append(carry, points[i])
			if points[i].t <= endAt {
				break
			}
		}
		return carry
	}

	// carry over data-points at times > endAt and the first data-points that is <= endAt
	p.external = carryOver(p.external)
	p.internal = carryOver(p.internal)

	// send events for all the data-points that were carried over
	evts := make([]events.Event, 0, len(p.external)+len(p.internal))
	iTWAP, eTWAP := twap(p.internal, p.startedAt, p.startedAt+duration), twap(p.external, p.startedAt, p.startedAt+duration)
	for _, dp := range p.external {
		evts = append(evts, events.NewFundingPeriodDataPointEvent(ctx, p.id, dp.price.String(), dp.t, p.seq, dataPointSourceExternal))
	}
	for _, dp := range p.internal {
		evts = append(evts, events.NewFundingPeriodDataPointEvent(ctx, p.id, dp.price.String(), dp.t, p.seq, dataPointSourceInternal))
	}
	// send event to say our new period has started
	p.broker.Send(events.NewFundingPeriodEvent(ctx, p.id, p.seq, p.startedAt, nil, nil, nil, ptr.From(iTWAP.String()), ptr.From(eTWAP.String())))
	if len(evts) > 0 {
		p.broker.SendBatch(evts)
	}
}

// addInternal adds an price point to our internal slice which represents a price value as seen by core.
func (p *Perpetual) addInternal(dp *dataPoint) error {
	if len(p.internal) > 0 && dp.t <= p.internal[len(p.internal)-1].t {
		// should not happen because these comes from ourselves, and if they are out of order somethings gone terribly wrong.
		p.log.Panic("internal settlement data-points received out of order")
	}
	p.internal = append(p.internal, dp)
	return nil
}

// addExternal adds an price point to our external slice which represents a price value as seen by and external data source.
func (p *Perpetual) addExternal(dp *dataPoint) error {
	// since the external points come in from the outside world there is no guarantee they'll come in order so
	// we put a little effort into making sure we insert it in the right place so that the data-points remain
	// ordered in time.

	// very first point, easy
	if len(p.external) == 0 {
		p.external = append(p.external, dp)
		return nil
	}

	// new point is later then our last, also easy
	last := p.external[len(p.external)-1]
	if last.t < dp.t {
		p.external = append(p.external, dp)
		return nil
	}

	// its before the first one, easy as well
	if dp.t < p.external[0].t {
		p.external = append([]*dataPoint{dp}, p.external...)
		return nil
	}

	// somewhere in the middle
	for i := len(p.external) - 1; i >= 0; i-- {
		data := p.external[i]

		if dp.t < data.t {
			// insert this point at position i - 1 then leave
			p.external = append(p.external, p.external[i-1:]...)
			p.external[i-1] = dp
			break
		}

		if dp.t == data.t {
			return ErrDataPointAlreadyExistsAtTime
		}
	}

	return nil
}

// readyForData returns whether not we are ready to start accepting data points.
func (p *Perpetual) readyForData() bool {
	return p.startedAt > 0
}

// haveData returns whether we have enough data to calculate a funding payment.
func (p *Perpetual) haveData(endAt int64) bool {
	if len(p.internal) == 0 || len(p.external) == 0 {
		return false
	}

	if p.internal[0].t > endAt || p.external[0].t > endAt {
		return false
	}

	return true
}

// calculateFundingPayment returns the funding payment and funding rate for the interval between when the current funding period
// started and the given time. Used on settlement-cues and for margin calculations.
func (p *Perpetual) calculateFundingPayment(internalTWAP, externalTWAP *num.Uint) (*num.Int, *num.Decimal) {
	p.log.Info("twap-calculations",
		logging.MarketID(p.id),
		logging.String("internal", internalTWAP.String()),
		logging.String("external", externalTWAP.String()),
	)

	// the funding payment is the difference between the two, the sign representing the direction of cash flow
	fundingPayment := num.IntFromUint(internalTWAP, true).Sub(num.IntFromUint(externalTWAP, true))
	fundingRate := num.DecimalFromInt(fundingPayment).Div(num.DecimalFromUint(externalTWAP))
	p.log.Info("funding payment calculated",
		logging.MarketID(p.id),
		logging.Uint64("seq", p.seq),
		logging.String("funding-payment", fundingPayment.String()),
		logging.String("funding-rate", fundingRate.String()))

	return fundingPayment, &fundingRate
}

// GetMarginIncrease returns the estimated extra margin required to account for the next funding payment
func (p *Perpetual) GetMarginIncrease(t int64) *num.Uint {
	// if we have no data, or the funding factor is zero, then the margin increase will always be zero
	if !p.haveData(t) || p.p.MarginFundingFactor.IsZero() {
		return num.UintZero()
	}
	// internal and external TWAP
	internalTWAP := twap(p.internal, p.startedAt, t)
	externalTWAP := twap(p.external, p.startedAt, t)
	fp, neg := internalTWAP.Delta(internalTWAP, externalTWAP)
	// if internal and external TWAP cancel eachother out, the margin increase will be zero
	// if internal TWAP > external TWAP, the margin increase would be a decrease, and has to be ignored
	if neg || fp.IsZero() {
		return num.UintZero()
	}
	// apply factor
	fpD := num.DecimalFromUint(fp).Mul(p.p.MarginFundingFactor)
	fp, _ = num.UintFromDecimal(fpD)
	return fp
}

// Calculates the twap of the given settlement data points over the given interval.
// The given set of points can extend beyond the interval [start, end] and any point
// lying outside that interval will be ignored.
func twap(points []*dataPoint, start, end int64) *num.Uint {
	if len(points) == 0 {
		return num.UintZero()
	}
	sum := num.UintZero()
	var prev *dataPoint
	for _, p := range points {
		// find the first point that is before or equal to the start of the interval
		if p.t <= start {
			prev = p
			continue
		}

		if p.t >= end {
			// this point is past the end time so we can stop now
			break
		}

		if prev != nil {
			tdiff := num.NewUint(uint64(p.t - num.MaxV(start, prev.t)))
			sum.Add(sum, num.UintZero().Mul(prev.price, tdiff))
		}
		prev = p
	}
	if prev == nil {
		return num.UintZero()
	}

	// process the final interval
	tdiff := num.NewUint(uint64(end - num.MaxV(start, prev.t)))
	sum.Add(sum, num.UintZero().Mul(prev.price, tdiff))

	return sum.Div(sum, num.NewUint(uint64(end-num.MaxV(start, points[0].t))))
}

func newPerpOracleBinding(p *types.Perps) (oracleBinding, error) {
	settleDataProp := strings.TrimSpace(p.DataSourceSpecBinding.SettlementDataProperty)
	settleScheduleProp := strings.TrimSpace(p.DataSourceSpecBinding.SettlementScheduleProperty)
	if len(settleDataProp) == 0 {
		return oracleBinding{}, errors.New("binding for settlement data cannot be blank")
	}
	if len(settleScheduleProp) == 0 {
		return oracleBinding{}, errors.New("binding for settlement schedule cannot be blank")
	}

	return oracleBinding{
		settlementDataProperty:     settleDataProp,
		settlementScheduleProperty: settleScheduleProp,
	}, nil
}
