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
	"time"

	"code.vegaprotocol.io/vega/core/datasource"
	dscommon "code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/datasource/spec"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/logging"
	"github.com/pkg/errors"
	"golang.org/x/exp/slices"

	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

var (
	year = num.DecimalFromInt64((24 * 365 * time.Hour).Nanoseconds())

	ErrDataPointAlreadyExistsAtTime = errors.New("data-point already exists at timestamp")
	ErrInitialPeriodNotStarted      = errors.New("initial settlement period not started")
)

type dataPointSource = eventspb.FundingPeriodDataPoint_Source

const (
	dataPointSourceExternal dataPointSource = eventspb.FundingPeriodDataPoint_SOURCE_EXTERNAL
	dataPointSourceInternal dataPointSource = eventspb.FundingPeriodDataPoint_SOURCE_INTERNAL
)

type fundingData struct {
	fundingPayment *num.Int
	fundingRate    num.Decimal
	internalTWAP   *num.Uint
	externalTWAP   *num.Uint
}

type cachedTWAP struct {
	log *logging.Logger

	periodStart int64        // the start of the funding period
	start       int64        // start of the TWAP period, which will be > periodStart if the first data-point comes after it
	end         int64        // time of the last calculated sub-product that was >= the last added data-point
	sumProduct  *num.Uint    // the sum-product of all the intervals between the data-points from `start` -> `end`
	points      []*dataPoint // the data-points used to calculate the twap
}

func NewCachedTWAP(log *logging.Logger, t int64) *cachedTWAP {
	return &cachedTWAP{
		log:         log,
		start:       t,
		periodStart: t,
		end:         t,
		sumProduct:  num.UintZero(),
	}
}

// setPeriod assigns the start and end of the calculated TWAP periods based on the time of incoming data-points.
// If the first data-point is before the true-period start it is still added but we ignore the contribution between
// data-point.t -> periodStart. So here we snap all values, then sanity check we've not set anything backwards.
func (c *cachedTWAP) setPeriod(start, end int64) {
	c.start = num.MaxV(c.periodStart, start)
	c.end = num.MaxV(c.periodStart, end)

	if c.end < c.start {
		c.log.Panic("twap interval has become backwards")
	}
}

// unwind returns the sum-product at the given time `t` where `t` is a time before the last
// data-point. We have to subtract each interval until we get to the first point where p.t < t,
// the index of `p` is also returned.
func (c *cachedTWAP) unwind(t int64) (*num.Uint, int) {
	if t < c.start {
		return num.UintZero(), 0
	}

	sumProduct := c.sumProduct.Clone()
	for i := len(c.points) - 1; i >= 0; i-- {
		point := c.points[i]
		prev := c.points[i-1]

		// now we need to remove the contribution from this interval
		delta := point.t - num.MaxV(prev.t, c.start)
		sub := num.UintZero().Mul(prev.price, num.NewUint(uint64(delta)))

		// before we subtract, lets sanity check some things
		if delta < 0 {
			c.log.Panic("twap data-points are out of order creating retrograde segment")
		}
		if sumProduct.LT(sub) {
			c.log.Panic("twap unwind is subtracting too much")
		}

		sumProduct.Sub(sumProduct, sub)

		if prev.t <= t {
			return sumProduct, i - 1
		}
	}

	c.log.Panic("have unwound to before initial data-point -- we shouldn't be here")
	return nil, 0
}

// calculate returns the TWAP at time `t` given the existing set of data-points. `t` can be
// any value and we will extend off the last-data-point if necessary, and also unwind intervals
// if the TWAP at a more historic time is required.
func (c *cachedTWAP) calculate(t int64) *num.Uint {
	if t < c.start || len(c.points) == 0 {
		return num.UintZero()
	}

	if t == c.end {
		// already have the sum product here, just twap-it
		return num.UintZero().Div(c.sumProduct, num.NewUint(uint64(c.end-c.start)))
	}

	// if the time we want the twap from is before the last data-point we need to unwind the intervals
	point := c.points[len(c.points)-1]
	if t < point.t {
		sumProduct, idx := c.unwind(t)
		p := c.points[idx]
		delta := t - num.MaxV(p.t, c.start)
		sumProduct.Add(sumProduct, num.UintZero().Mul(p.price, num.NewUint(uint64(delta))))
		return num.UintZero().Div(sumProduct, num.NewUint(uint64(t-c.start)))
	}

	// the twap we want is after the final data-point so we can just extend the calculation (or shortern if we've already extended)
	delta := t - c.end
	sumProduct := c.sumProduct.Clone()
	newPeriod := num.NewUint(uint64(t - c.start))
	lastPrice := point.price.Clone()

	// add or subtract from the sum-product based on if we are extending/shortening the interval
	switch {
	case delta < 0:
		sumProduct.Sub(sumProduct, lastPrice.Mul(lastPrice, num.NewUint(uint64(-delta))))
	case delta > 0:
		sumProduct.Add(sumProduct, lastPrice.Mul(lastPrice, num.NewUint(uint64(delta))))
	}
	// store these as the last calculated as its likely to be asked again
	c.setPeriod(c.start, t)
	c.sumProduct = sumProduct

	// now divide by the period to return the TWAP
	return num.UintZero().Div(sumProduct, newPeriod)
}

// insertPoint adds the given point (which is known to have arrived out of order) to
// the slice of points. The running sum-product is wound back to where we need to add
// the new point and then recalculated forwards to the point with the lastest timestamp.
func (c *cachedTWAP) insertPoint(point *dataPoint) (*num.Uint, error) {
	// unwind the intervals and set the end and sum-product to the unwound values
	sumProduct, idx := c.unwind(point.t)
	if c.points[idx].t == point.t {
		return nil, ErrDataPointAlreadyExistsAtTime
	}

	c.setPeriod(c.start, c.points[idx].t)
	c.sumProduct = sumProduct.Clone()

	// grab the data-points after the one we are inserting so that we can add them back in again
	subsequent := slices.Clone(c.points[idx+1:])
	c.points = c.points[:idx+1]

	// add the new point and calculate the TWAP
	twap := c.calculate(point.t)
	c.points = append(c.points, point)

	// now add the points that we unwound so that the running sum-product is amended
	// now that we've inserted the new point
	for _, p := range subsequent {
		c.calculate(p.t)
		c.points = append(c.points, p)
	}

	return twap, nil
}

// addPoint takes the given point and works out where it fits against what we already have, updates the
// running sum-product and returns the TWAP at point.t.
func (c *cachedTWAP) addPoint(point *dataPoint) (*num.Uint, error) {
	if len(c.points) == 0 || point.t < c.start {
		// first point, or new point is before the start of the funding period
		c.points = []*dataPoint{point}
		c.setPeriod(point.t, point.t)
		c.sumProduct = num.UintZero()
		return num.UintZero(), nil
	}

	// point to add is before the very first point we added, a little weird but ok
	if point.t <= c.points[0].t {
		points := c.points[:]
		c.points = []*dataPoint{point}
		c.setPeriod(point.t, point.t)
		c.sumProduct = num.UintZero()
		for _, p := range points {
			c.calculate(p.t)
			c.points = append(c.points, p)
		}
		return num.UintZero(), nil
	}

	// new point is after the last point, just calculate the TWAP at point.t and append
	// the new point to the slice
	lastPoint := c.points[len(c.points)-1]
	if point.t > lastPoint.t {
		twap := c.calculate(point.t)
		c.points = append(c.points, point)
		return twap, nil
	}

	if point.t == lastPoint.t {
		// already have a point for this time
		return nil, ErrDataPointAlreadyExistsAtTime
	}

	// we need to undo any extension past the last point we've done, we can do this by recalculating to the last point
	// which will remove the extension
	c.calculate(num.MaxV(c.start, lastPoint.t))

	// new point is before the last point, we need to unwind all the intervals and insert it into the correct place
	return c.insertPoint(point)
}

// A data-point that will be used to calculate periodic settlement in a perps market.
type dataPoint struct {
	// the asset price
	price *num.Uint
	// the timestamp of this data point
	t int64
}

// Perpetual represents a Perpetual as describe by the market framework.
type Perpetual struct {
	p   *types.Perps
	log *logging.Logger
	// oracle                 oracle
	settlementDataListener func(context.Context, *num.Numeric)
	broker                 Broker
	oracle                 scheduledOracle
	timeService            common.TimeService

	// id should be the same as the market id
	id string
	// enumeration of the settlement period so that we can track which points landed in each interval
	seq uint64
	// the time that this period interval started (in nanoseconds)
	startedAt int64
	// asset decimal places
	assetDP    uint32
	terminated bool

	// twap calculators
	internalTWAP *cachedTWAP
	externalTWAP *cachedTWAP
}

func (p Perpetual) GetCurrentPeriod() uint64 {
	return p.seq
}

func (p *Perpetual) Update(ctx context.Context, pp interface{}, oe OracleEngine) error {
	iPerp, ok := pp.(*types.InstrumentPerps)
	if !ok {
		p.log.Panic("attempting to update a perpetual into something else")
	}

	// unsubsribe all old oracles
	p.oracle.unsubAll(ctx)

	// grab all the new margin-factor and whatnot.
	p.p = iPerp.Perps

	// make sure we have all we need
	if p.p.DataSourceSpecForSettlementData == nil || p.p.DataSourceSpecForSettlementSchedule == nil || p.p.DataSourceSpecBinding == nil {
		return ErrDataSourceSpecAndBindingAreRequired
	}
	oracle, err := newPerpOracle(p.p)
	if err != nil {
		return err
	}

	// create specs from source
	osForSettle, err := spec.New(*datasource.SpecFromDefinition(*p.p.DataSourceSpecForSettlementData.Data))
	if err != nil {
		return err
	}
	osForSchedule, err := spec.New(*datasource.SpecFromDefinition(*p.p.DataSourceSpecForSettlementSchedule.Data))
	if err != nil {
		return err
	}
	if err = oracle.bindAll(ctx, oe, osForSettle, osForSchedule, p.receiveDataPoint, p.receiveSettlementCue); err != nil {
		return err
	}
	p.oracle = oracle // ensure oracle on perp is not an old copy

	return nil
}

func NewPerpetual(ctx context.Context, log *logging.Logger, p *types.Perps, marketID string, ts common.TimeService, oe OracleEngine, broker Broker, assetDP uint32) (*Perpetual, error) {
	// make sure we have all we need
	if p.DataSourceSpecForSettlementData == nil || p.DataSourceSpecForSettlementSchedule == nil || p.DataSourceSpecBinding == nil {
		return nil, ErrDataSourceSpecAndBindingAreRequired
	}
	oracle, err := newPerpOracle(p)
	if err != nil {
		return nil, err
	}
	// check decimal places for settlement data
	perp := &Perpetual{
		p:            p,
		id:           marketID,
		log:          log,
		timeService:  ts,
		broker:       broker,
		assetDP:      assetDP,
		externalTWAP: NewCachedTWAP(log, 0),
		internalTWAP: NewCachedTWAP(log, 0),
	}
	// create specs from source
	osForSettle, err := spec.New(*datasource.SpecFromDefinition(*p.DataSourceSpecForSettlementData.Data))
	if err != nil {
		return nil, err
	}
	osForSchedule, err := spec.New(*datasource.SpecFromDefinition(*p.DataSourceSpecForSettlementSchedule.Data))
	if err != nil {
		return nil, err
	}
	if err = oracle.bindAll(ctx, oe, osForSettle, osForSchedule, perp.receiveDataPoint, perp.receiveSettlementCue); err != nil {
		return nil, err
	}
	perp.oracle = oracle // ensure oracle on perp is not an old copy

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
	amount, neg := settlementData.Delta(settlementData, entryPriceInAsset)
	// Make sure net position is positive
	if netFractionalPosition.IsNegative() {
		netFractionalPosition = netFractionalPosition.Neg()
		neg = !neg
	}

	if p.log.IsDebug() {
		p.log.Debug("settlement",
			logging.String("entry-price-in-asset", entryPriceInAsset.String()),
			logging.String("settlement-data-in-asset", settlementData.String()),
			logging.String("net-fractional-position", netFractionalPosition.String()),
			logging.String("amount-in-decimal", netFractionalPosition.Mul(amount.ToDecimal()).String()),
			logging.String("amount-in-uint", amount.String()),
		)
	}
	a, rem := num.UintFromDecimalWithFraction(netFractionalPosition.Mul(amount.ToDecimal()))

	return &types.FinancialAmount{
		Asset:  p.p.SettlementAsset,
		Amount: a,
	}, neg, rem, nil
	// p.log.Panic("not implemented")
	// return nil, false, num.DecimalZero(), nil
}

// Value - returns the nominal value of a unit given a current mark price.
func (p *Perpetual) Value(markPrice *num.Uint) (*num.Uint, error) {
	return markPrice.Clone(), nil
}

// IsTradingTerminated - returns true when the oracle has signalled terminated market.
func (p *Perpetual) IsTradingTerminated() bool {
	return p.terminated
}

// GetAsset return the asset used by the future.
func (p *Perpetual) GetAsset() string {
	return p.p.SettlementAsset
}

func (p *Perpetual) UnsubscribeTradingTerminated(ctx context.Context) {
	// we could just use this call to indicate the underlying perp was terminted
	p.log.Info("unsubscribed trading data and cue oracle on perpetual termination", logging.String("quote-name", p.p.QuoteName))
	p.terminated = true
	p.oracle.unsubAll(ctx)
	p.handleSettlementCue(ctx, p.timeService.GetTimeNow().Truncate(time.Second).UnixNano())
}

func (p *Perpetual) UnsubscribeSettlementData(ctx context.Context) {
	p.log.Info("unsubscribed trading settlement data for", logging.String("quote-name", p.p.QuoteName))
	p.oracle.unsubAll(ctx)
}

func (p *Perpetual) OnLeaveOpeningAuction(ctx context.Context, t int64) {
	p.startedAt = t
	p.internalTWAP = NewCachedTWAP(p.log, t)
	p.externalTWAP = NewCachedTWAP(p.log, t)
	p.broker.Send(events.NewFundingPeriodEvent(ctx, p.id, p.seq, p.startedAt, nil, nil, nil, nil, nil))
}

// SubmitDataPoint this will add a data point produced internally by the core node.
func (p *Perpetual) SubmitDataPoint(ctx context.Context, price *num.Uint, t int64) error {
	if !p.readyForData() {
		return ErrInitialPeriodNotStarted
	}

	twap, err := p.internalTWAP.addPoint(&dataPoint{price: price.Clone(), t: t})
	if err != nil {
		return err
	}
	p.broker.Send(events.NewFundingPeriodDataPointEvent(ctx, p.id, price.String(), t, p.seq, dataPointSourceInternal, twap))
	return nil
}

func (p *Perpetual) receiveDataPoint(ctx context.Context, data dscommon.Data) error {
	if p.log.GetLevel() == logging.DebugLevel {
		p.log.Debug("new oracle data received", data.Debug()...)
	}

	settlDataDecimals := int64(p.oracle.binding.settlementDecimals)
	odata := &oracleData{
		settlData: &num.Numeric{},
	}
	switch p.oracle.binding.settlementType {
	case datapb.PropertyKey_TYPE_DECIMAL:
		settlDataAsDecimal, err := data.GetDecimal(p.oracle.binding.settlementProperty)
		if err != nil {
			p.log.Error(
				"could not parse decimal type property acting as settlement data",
				logging.Error(err),
			)
			return err
		}

		odata.settlData.SetDecimal(&settlDataAsDecimal)

	default:
		settlDataAsUint, err := data.GetUint(p.oracle.binding.settlementProperty)
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
	pTime, err := data.GetDataTimestampNano()
	if err != nil {
		p.log.Error("No timestamp associated with data point",
			logging.Error(err),
		)
		return err
	}

	// now add the price
	p.addExternalDataPoint(ctx, assetPrice, pTime)
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
		p.log.Debug("external data point for perpetual received before initial period", logging.String("id", p.id), logging.Int64("t", t))
		return
	}
	twap, err := p.externalTWAP.addPoint(&dataPoint{price: price.Clone(), t: t})
	if err != nil {
		p.log.Error("unable to add external data point",
			logging.String("id", p.id),
			logging.Error(err),
			logging.String("price", price.String()),
			logging.Int64("t", t))
		return
	}
	p.broker.Send(events.NewFundingPeriodDataPointEvent(ctx, p.id, price.String(), t, p.seq, dataPointSourceExternal, twap))
}

func (p *Perpetual) receiveSettlementCue(ctx context.Context, data dscommon.Data) error {
	if p.log.GetLevel() == logging.DebugLevel {
		p.log.Debug("new schedule oracle data received", data.Debug()...)
	}
	t, err := data.GetTimestamp(p.oracle.binding.scheduleProperty)
	if err != nil {
		p.log.Error("schedule data not valid", data.Debug()...)
		return err
	}

	// the internal cue gives us the time in seconds, so convert to nanoseconds
	t = time.Unix(t, 0).UnixNano()

	p.handleSettlementCue(ctx, t)
	if p.log.GetLevel() == logging.DebugLevel {
		p.log.Debug("perp schedule trigger processed")
	}
	return nil
}

// handleSettlementCue will be hooked up as a subscriber to the oracle data for the notification that the settlement period has ended.
func (p *Perpetual) handleSettlementCue(ctx context.Context, t int64) {
	if !p.readyForData() {
		if p.log.GetLevel() == logging.DebugLevel {
			p.log.Debug("first funding period not started -- ignoring settlement-cue")
		}
		return
	}

	if !p.haveDataBeforeGivenTime(t) || t == p.startedAt {
		// we have no points, or the interval is zero length so we just start a new interval
		p.broker.Send(events.NewFundingPeriodEvent(ctx, p.id, p.seq, p.startedAt, ptr.From(t), nil, nil, nil, nil))
		p.startNewFundingPeriod(ctx, t)
		return
	}

	// do the calculation
	r := p.calculateFundingPayment(t)

	// send it away!
	fp := &num.Numeric{}
	p.settlementDataListener(ctx, fp.SetInt(r.fundingPayment))

	// now restart the interval
	p.broker.Send(events.NewFundingPeriodEvent(ctx, p.id, p.seq, p.startedAt, ptr.From(t),
		ptr.From(r.fundingPayment.String()),
		ptr.From(r.fundingRate.String()),
		ptr.From(r.internalTWAP.String()),
		ptr.From(r.externalTWAP.String())),
	)
	p.startNewFundingPeriod(ctx, t)
}

func (p *Perpetual) GetData(t int64) *types.ProductData {
	if !p.readyForData() || !p.haveData() {
		return nil
	}

	r := p.calculateFundingPayment(t)
	return &types.ProductData{
		Data: &types.PerpetualData{
			FundingPayment: r.fundingPayment.String(),
			FundingRate:    r.fundingRate.String(),
			ExternalTWAP:   r.externalTWAP.String(),
			InternalTWAP:   r.internalTWAP.String(),
		},
	}
}

// restarts the funcing period at time st.
func (p *Perpetual) startNewFundingPeriod(ctx context.Context, endAt int64) {
	if p.terminated {
		// the perpetual market has been terminated so we won't start a new funding period
		return
	}

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
	external := carryOver(p.externalTWAP.points)
	internal := carryOver(p.internalTWAP.points)

	// new period new life
	p.externalTWAP = NewCachedTWAP(p.log, endAt)
	p.internalTWAP = NewCachedTWAP(p.log, endAt)

	// send events for all the data-points that were carried over
	evts := make([]events.Event, 0, len(external)+len(internal))
	iTWAP, eTWAP := num.UintZero(), num.UintZero()
	for _, dp := range external {
		eTWAP, _ := p.externalTWAP.addPoint(dp)
		evts = append(evts, events.NewFundingPeriodDataPointEvent(ctx, p.id, dp.price.String(), dp.t, p.seq, dataPointSourceExternal, eTWAP))
	}
	for _, dp := range internal {
		iTWAP, _ := p.internalTWAP.addPoint(dp)
		evts = append(evts, events.NewFundingPeriodDataPointEvent(ctx, p.id, dp.price.String(), dp.t, p.seq, dataPointSourceInternal, iTWAP))
	}
	// send event to say our new period has started
	p.broker.Send(events.NewFundingPeriodEvent(ctx, p.id, p.seq, p.startedAt, nil, nil, nil, ptr.From(iTWAP.String()), ptr.From(eTWAP.String())))
	if len(evts) > 0 {
		p.broker.SendBatch(evts)
	}
}

// readyForData returns whether not we are ready to start accepting data points.
func (p *Perpetual) readyForData() bool {
	return p.startedAt > 0
}

// haveDataBeforeGivenTime returns whether we have at least one data point from each of the internal and external price series before the given time.
func (p *Perpetual) haveDataBeforeGivenTime(endAt int64) bool {
	if !p.readyForData() {
		return false
	}

	if !p.haveData() {
		return false
	}

	if p.internalTWAP.points[0].t > endAt || p.externalTWAP.points[0].t > endAt {
		return false
	}

	return true
}

// haveData returns whether we have at least one data point from each of the internal and external price series.
func (p *Perpetual) haveData() bool {
	return len(p.internalTWAP.points) > 0 && len(p.externalTWAP.points) > 0
}

// calculateFundingPayment returns the funding payment and funding rate for the interval between when the current funding period
// started and the given time. Used on settlement-cues and for margin calculations.
func (p *Perpetual) calculateFundingPayment(t int64) *fundingData {
	internalTWAP := p.internalTWAP.calculate(t)
	externalTWAP := p.externalTWAP.calculate(t)

	if p.log.GetLevel() == logging.DebugLevel {
		p.log.Debug("twap-calculations",
			logging.MarketID(p.id),
			logging.String("internal", internalTWAP.String()),
			logging.String("external", externalTWAP.String()),
		)
	}

	// the funding payment is the difference between the two, the sign representing the direction of cash flow
	fundingPayment := num.IntFromUint(internalTWAP, true).Sub(num.IntFromUint(externalTWAP, true))

	// apply interest-rates if necessary
	if !p.p.InterestRate.IsZero() {
		delta := t - p.startedAt
		if p.log.GetLevel() == logging.DebugLevel {
			p.log.Debug("applying interest-rate with clamping", logging.String("funding-payment", fundingPayment.String()), logging.Int64("delta", delta))
		}
		fundingPayment.Add(p.calculateInterestTerm(externalTWAP, internalTWAP, delta))
	}

	fundingRate := num.DecimalZero()
	if !externalTWAP.IsZero() {
		fundingRate = num.DecimalFromInt(fundingPayment).Div(num.DecimalFromUint(externalTWAP))
	}
	if p.log.GetLevel() == logging.DebugLevel {
		p.log.Debug("funding payment calculated",
			logging.MarketID(p.id),
			logging.Uint64("seq", p.seq),
			logging.String("funding-payment", fundingPayment.String()),
			logging.String("funding-rate", fundingRate.String()))
	}
	return &fundingData{
		fundingPayment: fundingPayment,
		fundingRate:    fundingRate,
		externalTWAP:   externalTWAP,
		internalTWAP:   internalTWAP,
	}
}

func (p *Perpetual) calculateInterestTerm(externalTWAP, internalTWAP *num.Uint, delta int64) *num.Int {
	// get delta in terms of years
	td := num.DecimalFromInt64(delta).Div(year)

	// convert into num types we need
	sTWAP := num.DecimalFromUint(externalTWAP)
	fTWAP := num.DecimalFromUint(internalTWAP)

	// interest = (1 + r * td) * s_swap - f_swap
	interest := num.DecimalOne().Add(p.p.InterestRate.Mul(td)).Mul(sTWAP)
	interest = interest.Sub(fTWAP)

	upperBound := num.DecimalFromUint(externalTWAP).Mul(p.p.ClampUpperBound)
	lowerBound := num.DecimalFromUint(externalTWAP).Mul(p.p.ClampLowerBound)

	clampedInterest := num.MinD(upperBound, num.MaxD(lowerBound, interest))
	if p.log.GetLevel() == logging.DebugLevel {
		p.log.Debug("clamped interest and bounds",
			logging.MarketID(p.id),
			logging.String("lower-bound", p.p.ClampLowerBound.String()),
			logging.String("interest", interest.String()),
			logging.String("upper-bound", p.p.ClampUpperBound.String()),
			logging.String("clamped-interest", clampedInterest.String()),
		)
	}

	result, overflow := num.IntFromDecimal(clampedInterest)
	if overflow {
		p.log.Panic("overflow converting interest term to Int", logging.String("clampedInterest", clampedInterest.String()))
	}
	return result
}

// GetMarginIncrease returns the estimated extra margin required to account for the next funding payment
// for a party with a position of +1.
func (p *Perpetual) GetMarginIncrease(t int64) num.Decimal {
	// if we have no data, or the funding factor is zero, then the margin increase will always be zero
	if !p.haveDataBeforeGivenTime(t) || p.p.MarginFundingFactor.IsZero() {
		return num.DecimalZero()
	}

	fundingPayment := p.calculateFundingPayment(t).fundingPayment

	// apply factor
	return num.DecimalFromInt(fundingPayment).Mul(p.p.MarginFundingFactor)
}
