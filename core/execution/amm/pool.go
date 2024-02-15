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

package amm

import (
	"code.vegaprotocol.io/vega/core/positions"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

// ephemeralPosition keeps track of the pools position as if its generated orders had traded.
type ephemeralPosition struct {
	size         int64
	averageEntry *num.Uint
}

type curve struct {
	l    *num.Uint   // virtual liquidity
	high *num.Uint   // high price value, upper bound if upper curve, base price is lower curve
	low  *num.Uint   // low price value, base price if upper curve, lower bound if lower curve
	rf   num.Decimal // commitment scaling factor
}

type Pool struct {
	ID         string
	SubAccount string
	Commitment *num.Uint
	Parameters *types.ConcentratedLiquidityParameters

	asset          string
	market         string
	party          string
	collateral     Collateral
	position       Position
	priceFactor    *num.Uint
	positionFactor num.Decimal

	// current pool status
	status types.AMMPoolStatus

	// sqrt function to use.
	sqrt sqrtFn

	// the two curves joined at base-price used to determine price and volume in the pool
	// lower is used when the pool is long.
	lower *curve
	upper *curve

	// during the matching process across price levels we need to keep tracking of the pools potential positions
	// as if those matching orders were to trade. This is so that when we generate more orders at the next price level
	// for the same incoming order, the second round of generated orders are priced as if the first round had traded.
	eph *ephemeralPosition

	// one price tick
	oneTick *num.Uint
}

func NewPool(
	id,
	subAccount,
	asset string,
	submit *types.SubmitAMM,
	sqrt sqrtFn,
	collateral Collateral,
	position Position,
	rf *types.RiskFactor,
	sf *types.ScalingFactors,
	linearSlippage num.Decimal,
	priceFactor *num.Uint,
	positionFactor num.Decimal,
) *Pool {
	pool := &Pool{
		ID:             id,
		SubAccount:     subAccount,
		Commitment:     submit.CommitmentAmount,
		Parameters:     submit.Parameters,
		market:         submit.MarketID,
		party:          submit.Party,
		asset:          asset,
		sqrt:           sqrt,
		collateral:     collateral,
		position:       position,
		priceFactor:    priceFactor,
		positionFactor: positionFactor,
		oneTick:        num.UintZero().Mul(num.UintOne(), priceFactor),
		status:         types.AMMPoolStatusActive,
	}
	pool.setCurves(rf, sf, linearSlippage)
	return pool
}

func NewPoolFromProto(
	sqrt sqrtFn,
	collateral Collateral,
	position Position,
	state *snapshotpb.PoolMapEntry_Pool,
	party string,
	priceFactor *num.Uint,
) *Pool {
	return &Pool{
		ID:         state.Id,
		SubAccount: state.SubAccount,
		Commitment: num.MustUintFromString(state.Commitment, 10),
		Parameters: &types.ConcentratedLiquidityParameters{
			Base:                    num.MustUintFromString(state.Parameters.Base, 10),
			LowerBound:              num.MustUintFromString(state.Parameters.LowerBound, 10),
			UpperBound:              num.MustUintFromString(state.Parameters.UpperBound, 10),
			MarginRatioAtLowerBound: ptr.From(num.MustDecimalFromString(state.Parameters.MarginRatioAtLowerBound)),
			MarginRatioAtUpperBound: ptr.From(num.MustDecimalFromString(state.Parameters.MarginRatioAtUpperBound)),
		},
		party:      party,
		market:     state.Market,
		asset:      state.Asset,
		sqrt:       sqrt,
		collateral: collateral,
		position:   position,
		lower: &curve{
			l:    num.MustUintFromString(state.Lower.L, 10),
			high: num.MustUintFromString(state.Lower.High, 10),
			low:  num.MustUintFromString(state.Lower.Low, 10),
			rf:   num.MustDecimalFromString(state.Lower.Rf),
		},
		upper: &curve{
			l:    num.MustUintFromString(state.Upper.L, 10),
			high: num.MustUintFromString(state.Upper.High, 10),
			low:  num.MustUintFromString(state.Upper.Low, 10),
			rf:   num.MustDecimalFromString(state.Upper.Rf),
		},
		priceFactor: priceFactor,
		oneTick:     num.UintZero().Mul(num.UintOne(), priceFactor),
		status:      state.Status,
	}
}

func (p *Pool) IntoProto() *snapshotpb.PoolMapEntry_Pool {
	return &snapshotpb.PoolMapEntry_Pool{
		Id:         p.ID,
		SubAccount: p.SubAccount,
		Commitment: p.Commitment.String(),
		Parameters: p.Parameters.ToProtoEvent(),
		Market:     p.market,
		Asset:      p.asset,
		Lower: &snapshotpb.PoolMapEntry_Curve{
			L:    p.lower.l.String(),
			High: p.lower.high.String(),
			Low:  p.lower.low.String(),
			Rf:   p.lower.rf.String(),
		},
		Upper: &snapshotpb.PoolMapEntry_Curve{
			L:    p.upper.l.String(),
			High: p.upper.high.String(),
			Low:  p.upper.low.String(),
			Rf:   p.upper.rf.String(),
		},
		Status: p.status,
	}
}

func (p *Pool) Update(
	amend *types.AmendAMM,
	rf *types.RiskFactor,
	sf *types.ScalingFactors,
	linearSlippage num.Decimal,
) {
	p.Commitment = amend.CommitmentAmount
	p.Parameters.ApplyUpdate(amend.Parameters)
	p.setCurves(rf, sf, linearSlippage)
}

// generateCurve creates the curve details and calculates its virtual liquidity.
func generateCurve(
	sqrt sqrtFn,
	commitment,
	low, high *num.Uint,
	p *num.Uint,
	riskFactor,
	marginFactor,
	linearSlippage num.Decimal,
	marginRatio *num.Decimal,
	positionFactor num.Decimal,
) *curve {
	// rf = 1 / ( mf * ( risk-factor + slippage ) )
	rf := num.DecimalOne().Div(marginFactor.Mul(riskFactor.Add(linearSlippage)))
	if marginRatio != nil {
		// rf = min(rf, 1/margin-ratio)
		rf = num.MinD(rf, num.DecimalOne().Div(*marginRatio))
	}

	// we scale rf by the position factor since that is used to calculate the theoretical volume (pv) at the boundary
	// just here below, and also when calculating the fair price when the pool is in a short position.
	rf = rf.Mul(positionFactor)

	// calculate the theoretical volume at the extreme i.e upper-bound for high curve, lower bound for low curve
	// pv = rf * commitment / p
	pv := rf.Mul(commitment.ToDecimal()).Div(p.ToDecimal())

	// pv * sqrt(high) * sqrt(low)
	term1 := pv.Mul(sqrt(high).Mul(sqrt(low)))

	// sqrt(high) - sqrt(low)
	term2 := sqrt(high).Sub(sqrt(low))

	// L = pv * sqrt(high) * sqrt(low) / ( sqrt(high) - sqrt(low) )
	l := term1.Div(term2)
	ld, _ := num.UintFromDecimal(l)
	return &curve{
		l:    ld,
		rf:   rf,
		low:  low,
		high: high,
	}
}

func (p *Pool) setCurves(
	rfs *types.RiskFactor,
	sfs *types.ScalingFactors,
	linearSlippage num.Decimal,
) {
	// convert the bounds into asset precision
	lowerBound := num.UintZero().Mul(p.Parameters.LowerBound, p.priceFactor)
	base := num.UintZero().Mul(p.Parameters.Base, p.priceFactor)
	upperBound := num.UintZero().Mul(p.Parameters.UpperBound, p.priceFactor)

	p.lower = generateCurve(
		p.sqrt,
		p.Commitment.Clone(),
		lowerBound,
		base,
		lowerBound,
		rfs.Long,
		sfs.InitialMargin,
		linearSlippage,
		p.Parameters.MarginRatioAtLowerBound,
		p.positionFactor,
	)

	p.upper = generateCurve(
		p.sqrt,
		p.Commitment.Clone(),
		base.Clone(),
		upperBound,
		upperBound,
		rfs.Short,
		sfs.InitialMargin,
		linearSlippage,
		p.Parameters.MarginRatioAtUpperBound,
		p.positionFactor,
	)
}

// impliedPosition returns the position of the pool if its fair-price were the given price. `l` is
// the virtual liquidity of the pool, and `sqrtPrice` and `sqrtHigh` are, the square-roots of the
// price to calculate the position for, and higher boundary of the curve.
func impliedPosition(sqrtPrice, sqrtHigh num.Decimal, l *num.Uint) *num.Uint {
	// L * (sqrt(high) - sqrt(price))
	numer := sqrtHigh.Sub(sqrtPrice).Mul(l.ToDecimal())

	// sqrt(high) * sqrt(price)
	denom := sqrtHigh.Mul(sqrtPrice)

	// L * (sqrt(high) - sqrt(price)) / sqrt(high) * sqrt(price)
	res, _ := num.UintFromDecimal(numer.Div(denom))
	return res
}

// OrderbookShape returns slices of virtual buy and sell orders that the AMM has over a given range
// and is essentially a view on the AMM's personal order-book.
func (p *Pool) OrderbookShape(from, to *num.Uint) ([]*types.Order, []*types.Order) {

	buys := []*types.Order{}
	sells := []*types.Order{}

	if from == nil {
		from = p.lower.low
	}
	if to == nil {
		to = p.upper.high
	}

	// any volume strictly below the fair price will be a buy, and volume above will be a sell
	side := types.SideBuy
	fairPrice := p.fairPrice()

	ordersFromCurve := func(cu *curve, from, to *num.Uint) {

		from = num.Max(from, cu.low)
		to = num.Min(to, cu.high)
		price := from
		for price.LT(to) {

			next := num.UintZero().AddSum(price, p.oneTick)
			volume, _ := num.UintZero().Delta(
				impliedPosition(p.sqrt(price), p.sqrt(cu.high), cu.l),
				impliedPosition(p.sqrt(next), p.sqrt(cu.high), cu.l),
			)

			if side == types.SideBuy && next.GT(fairPrice) {
				// now switch to sells, we're over the fair-price now
				side = types.SideSell
			}

			order := &types.Order{
				Size:  volume.Uint64(),
				Side:  side,
				Price: price.Clone(),
			}

			if side == types.SideBuy {
				buys = append(buys, order)
			} else {
				sells = append(sells, order)
			}

			price = next
		}

	}
	ordersFromCurve(p.lower, from, to)
	ordersFromCurve(p.upper, from, to)
	return buys, sells
}

// VolumeBetweenPrices returns the volume the pool is willing to provide between the two given price levels for side of a given order
// that is trading with the pool. If `nil` is provided for either price then we take the full volume in that direction.
func (p *Pool) VolumeBetweenPrices(side types.Side, price1 *num.Uint, price2 *num.Uint) uint64 {
	pos, _ := p.getPosition()
	st, nd := price1, price2

	// get the curve based on the pool's current position, if the position is zero we take the curve the trade will put us in
	// e.g trading with an incoming buy order will make the pool short, so we take the upper curve.
	var cu *curve
	if pos < 0 || (pos == 0 && side == types.SideBuy) {
		cu = p.upper
	} else {
		cu = p.lower
	}

	if price1 == nil {
		st = cu.low
	}

	if price2 == nil {
		nd = cu.high
	}

	if st.EQ(nd) {
		return 0
	}

	if st.GT(nd) {
		st, nd = nd, st
	}

	// there is no volume outside of the bounds for the curve so we snap st, nd to the boundaries
	st = num.Max(st, cu.low)
	nd = num.Min(nd, cu.high)
	if st.GTE(nd) {
		return 0
	}

	// abs(P(st) - P(nd))
	volume, _ := num.UintZero().Delta(
		impliedPosition(p.sqrt(st), p.sqrt(cu.high), cu.l),
		impliedPosition(p.sqrt(nd), p.sqrt(cu.high), cu.l),
	)

	if p.closing() {
		return num.MinV(volume.Uint64(), uint64(num.AbsV(pos)))
	}

	return volume.Uint64()
}

// getBalance returns the total balance of the pool i.e it's general account + it's margin account.
func (p *Pool) getBalance() *num.Uint {
	general, err := p.collateral.GetPartyGeneralAccount(p.SubAccount, p.asset)
	if err != nil {
		panic("general account not created")
	}

	margin, _ := p.collateral.GetPartyMarginAccount(p.market, p.SubAccount, p.asset)
	if err != nil {
		panic("margin account not created")
	}

	return num.UintZero().AddSum(general.Balance, margin.Balance)
}

// setEphemeralPosition is called when we are starting the matching process against this pool
// so that we can track its position and average-entry as it goes through the matching process.
func (p *Pool) setEphemeralPosition() {
	if p.eph != nil {
		return
	}
	p.eph = &ephemeralPosition{
		size:         0,
		averageEntry: num.UintZero(),
	}

	if pos := p.position.GetPositionsByParty(p.SubAccount); len(pos) != 0 {
		p.eph.size = pos[0].Size()
		p.eph.averageEntry = pos[0].AverageEntryPrice()
	}
}

// updateEphemeralPosition sets the pools transient position given a generated order.
func (p *Pool) updateEphemeralPosition(order *types.Order) {
	if order.Side == types.SideSell {
		p.eph.averageEntry = positions.CalcVWAP(p.eph.averageEntry, -p.eph.size, int64(order.Size), order.Price)
		p.eph.size -= int64(order.Size)
		return
	}

	p.eph.averageEntry = positions.CalcVWAP(p.eph.averageEntry, p.eph.size, int64(order.Size), order.Price)
	p.eph.size += int64(order.Size)
}

// clearEphemeralPosition signifies that the matching process has finished
// and the pool can continue to read it's position from the positions engine.
func (p *Pool) clearEphemeralPosition() {
	p.eph = nil
}

// getPosition gets the pools current position an average-entry price.
func (p *Pool) getPosition() (int64, *num.Uint) {
	if p.eph != nil {
		return p.eph.size, p.eph.averageEntry.Clone()
	}

	if pos := p.position.GetPositionsByParty(p.SubAccount); len(pos) != 0 {
		return pos[0].Size(), pos[0].AverageEntryPrice()
	}
	return 0, num.UintZero()
}

// virtualBalancesLong returns the pools x, y balances when the pool has a negative position, where
//
// x = P + (cc * rf) / sqrt(pl) + L / sqrt(pl),
// y = abs(P) * average-entry + L * sqrt(pl).
func (p *Pool) virtualBalancesShort(pos int64, ae *num.Uint) (num.Decimal, num.Decimal) {
	cu := p.upper
	balance := p.getBalance()

	// lets start with x

	// P
	term1x := num.DecimalFromInt64(-pos)

	// cc * rf / pu
	term2x := cu.rf.Mul(num.DecimalFromUint(balance)).Div(num.DecimalFromUint(cu.high))

	// L / sqrt(pl)
	term3x := cu.l.ToDecimal().Div(p.sqrt(cu.high))

	// x = P + (cc * rf / pu) + (L / sqrt(pl))
	x := term2x.Add(term3x).Sub(term1x)

	// now lets get y

	// abs(P) * average-entry
	term1y := ae.Mul(ae, num.NewUint(uint64(-pos)))

	// L * sqrt(pl)
	term2y := cu.l.ToDecimal().Mul(p.sqrt(cu.low))

	// y = abs(P) * average-entry + L * pl
	y := term1y.ToDecimal().Add(term2y)
	return x, y
}

// virtualBalancesLong returns the pools x, y balances when the pool has a positive position, where
//
// x = P + (L / sqrt(pu)),
// y = L * (sqrt(pu) - sqrt(pl)) - P * average-entry + (L * sqrt(pl)).
func (p *Pool) virtualBalancesLong(pos int64, ae *num.Uint) (num.Decimal, num.Decimal) {
	cu := p.lower
	// lets start with x

	// P
	term1x := num.DecimalFromInt64(pos)

	// L / sqrt(pu)
	term2x := cu.l.ToDecimal().Div(p.sqrt(cu.high))
	x := term1x.Add(term2x)

	// now lets move to y

	// L * (sqrt(pu) - sqrt(pl)) + (L * sqrt(pl)) => L * sqrt(pu)
	term1y := cu.l.ToDecimal().Mul(p.sqrt(cu.high))

	// P * average-entry
	term2y := ae.Mul(ae, num.NewUint(uint64(pos)))

	y := term1y.Sub(term2y.ToDecimal())
	return x, y
}

// fairPrice returns the fair price of the pool given its current position.
func (p *Pool) fairPrice() *num.Uint {
	pos, ae := p.getPosition()
	if pos == 0 {
		return p.lower.high.Clone()
	}

	x, y := p.virtualBalances(pos, ae, types.SideUnspecified)
	fairPrice, _ := num.UintFromDecimal(y.Div(x))
	return fairPrice
}

// virtualBalances returns the pools x, y values where x is the balance in contracts and y is the balance in asset.
func (p *Pool) virtualBalances(pos int64, ae *num.Uint, side types.Side) (num.Decimal, num.Decimal) {
	switch {
	case pos < 0:
		return p.virtualBalancesShort(pos, ae)
	case pos > 0:
		return p.virtualBalancesLong(pos, ae)
	case side == types.SideBuy:
		// zero position but incoming is buy which will make pool short
		return p.virtualBalancesShort(pos, ae)
	case side == types.SideSell:
		// zero position but incoming is sell which will make pool long
		return p.virtualBalancesLong(pos, ae)
	default:
		panic("should not reach here")
	}
}

// BestPrice returns the price that the pool is willing to trade for the given order side.
func (p *Pool) BestPrice(order *types.Order) *num.Uint {
	fairPrice := p.fairPrice()
	switch {
	case order == nil:
		// special case where we've been asked for a fair price
		return fairPrice
	case order.Side == types.SideBuy:
		// incoming is a buy, so we +1 to the fair price
		return fairPrice.AddSum(p.oneTick)
	case order.Side == types.SideSell:
		// incoming is a sell so we - 1 the fair price
		return fairPrice.Sub(fairPrice, p.oneTick)
	default:
		panic("should never reach here")
	}
}

func (p *Pool) closing() bool {
	return p.status == types.AMMPoolStatusReduceOnly
}

func (p *Pool) canTrade(order *types.Order) bool {
	if !p.closing() {
		return true
	}

	pos, _ := p.getPosition()
	// pool is long incoming order is a buy and will make it shorter, its ok
	if pos > 0 && order.Side == types.SideBuy {
		return true
	}
	if pos < 0 && order.Side == types.SideSell {
		return true
	}
	return false
}
