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
	"fmt"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

// ephemeralPosition keeps track of the pools position as if its generated orders had traded.
type ephemeralPosition struct {
	size int64
}

type curve struct {
	l     num.Decimal // virtual liquidity
	high  *num.Uint   // high price value, upper bound if upper curve, base price is lower curve
	low   *num.Uint   // low price value, base price if upper curve, lower bound if lower curve
	empty bool        // if true the curve is of zero length and represents no liquidity on this side of the amm

	// the theoretical position of the curve at its lower boundary
	// note that this equals Vega's position at the boundary only in the lower curve, since Vega position == curve-position
	// in the upper curve Vega's position == 0 => position of `pv`` in curve-position, Vega's position pv => 0 in curve-position
	pv num.Decimal
}

func (c *curve) volumeBetweenPrices(sqrt sqrtFn, st, nd *num.Uint) uint64 {
	if c.l.IsZero() || c.empty {
		return 0
	}

	st = num.Max(st, c.low)
	nd = num.Min(nd, c.high)

	if st.GTE(nd) {
		return 0
	}

	// abs(P(st) - P(nd))
	volume, _ := num.UintZero().Delta(
		impliedPosition(sqrt(st), sqrt(c.high), c.l),
		impliedPosition(sqrt(nd), sqrt(c.high), c.l),
	)
	return volume.Uint64()
}

type Pool struct {
	ID          string
	SubAccount  string
	Commitment  *num.Uint
	ProposedFee num.Decimal
	Parameters  *types.ConcentratedLiquidityParameters

	asset          string
	market         string
	party          string
	collateral     Collateral
	position       Position
	priceFactor    num.Decimal
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
	priceFactor num.Decimal,
	positionFactor num.Decimal,
) (*Pool, error) {
	oneTick, _ := num.UintFromDecimal(num.DecimalOne().Mul(priceFactor))
	pool := &Pool{
		ID:             id,
		SubAccount:     subAccount,
		Commitment:     submit.CommitmentAmount,
		ProposedFee:    submit.ProposedFee,
		Parameters:     submit.Parameters,
		market:         submit.MarketID,
		party:          submit.Party,
		asset:          asset,
		sqrt:           sqrt,
		collateral:     collateral,
		position:       position,
		priceFactor:    priceFactor,
		positionFactor: positionFactor,
		oneTick:        oneTick,
		status:         types.AMMPoolStatusActive,
	}
	err := pool.setCurves(rf, sf, linearSlippage)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

func NewPoolFromProto(
	sqrt sqrtFn,
	collateral Collateral,
	position Position,
	state *snapshotpb.PoolMapEntry_Pool,
	party string,
	priceFactor num.Decimal,
) (*Pool, error) {
	oneTick, _ := num.UintFromDecimal(num.DecimalOne().Mul(priceFactor))

	var lowerLeverage, upperLeverage *num.Decimal
	if state.Parameters.LeverageAtLowerBound != nil {
		l, err := num.DecimalFromString(*state.Parameters.LeverageAtLowerBound)
		if err != nil {
			return nil, err
		}
		lowerLeverage = &l
	}
	if state.Parameters.LeverageAtUpperBound != nil {
		l, err := num.DecimalFromString(*state.Parameters.LeverageAtUpperBound)
		if err != nil {
			return nil, err
		}
		upperLeverage = &l
	}

	base, overflow := num.UintFromString(state.Parameters.Base, 10)
	if overflow {
		return nil, fmt.Errorf("failed to convert string to Uint: %s", state.Parameters.Base)
	}

	lower, overflow := num.UintFromString(state.Parameters.LowerBound, 10)
	if overflow {
		return nil, fmt.Errorf("failed to convert string to Uint: %s", state.Parameters.LowerBound)
	}

	upper, overflow := num.UintFromString(state.Parameters.UpperBound, 10)
	if overflow {
		return nil, fmt.Errorf("failed to convert string to Uint: %s", state.Parameters.UpperBound)
	}

	upperCu, err := NewCurveFromProto(state.Upper)
	if err != nil {
		return nil, err
	}

	lowerCu, err := NewCurveFromProto(state.Lower)
	if err != nil {
		return nil, err
	}

	return &Pool{
		ID:         state.Id,
		SubAccount: state.SubAccount,
		Commitment: num.MustUintFromString(state.Commitment, 10),
		Parameters: &types.ConcentratedLiquidityParameters{
			Base:                 base,
			LowerBound:           lower,
			UpperBound:           upper,
			LeverageAtLowerBound: lowerLeverage,
			LeverageAtUpperBound: upperLeverage,
		},
		party:       party,
		market:      state.Market,
		asset:       state.Asset,
		sqrt:        sqrt,
		collateral:  collateral,
		position:    position,
		lower:       lowerCu,
		upper:       upperCu,
		priceFactor: priceFactor,
		oneTick:     oneTick,
		status:      state.Status,
	}, nil
}

func NewCurveFromProto(c *snapshotpb.PoolMapEntry_Curve) (*curve, error) {
	l, err := num.DecimalFromString(c.L)
	if err != nil {
		return nil, err
	}

	pv, err := num.DecimalFromString(c.Pv)
	if err != nil {
		return nil, err
	}

	high, overflow := num.UintFromString(c.High, 10)
	if overflow {
		return nil, fmt.Errorf("failed to convert string to Uint: %s", c.High)
	}

	low, overflow := num.UintFromString(c.Low, 10)
	if overflow {
		return nil, fmt.Errorf("failed to convert string to Uint: %s", c.Low)
	}
	return &curve{
		l:     l,
		high:  high,
		low:   low,
		empty: c.Empty,
		pv:    pv,
	}, nil
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
			L:     p.lower.l.String(),
			High:  p.lower.high.String(),
			Low:   p.lower.low.String(),
			Empty: p.lower.empty,
			Pv:    p.lower.pv.String(),
		},
		Upper: &snapshotpb.PoolMapEntry_Curve{
			L:     p.upper.l.String(),
			High:  p.upper.high.String(),
			Low:   p.upper.low.String(),
			Empty: p.upper.empty,
			Pv:    p.lower.pv.String(),
		},
		Status: p.status,
	}
}

// Update returns a copy of the give pool but with its curves and parameters update as specified by `amend`.
func (p *Pool) Update(
	amend *types.AmendAMM,
	rf *types.RiskFactor,
	sf *types.ScalingFactors,
	linearSlippage num.Decimal,
) (*Pool, error) {
	commitment := p.Commitment.Clone()
	if amend.CommitmentAmount != nil {
		commitment = amend.CommitmentAmount
	}

	proposedFee := p.ProposedFee
	if amend.ProposedFee.IsPositive() {
		proposedFee = amend.ProposedFee
	}

	// parameters cannot only be updated all at once or not at all
	parameters := p.Parameters.Clone()
	if amend.Parameters != nil {
		parameters = amend.Parameters
	}

	updated := &Pool{
		ID:             p.ID,
		SubAccount:     p.SubAccount,
		Commitment:     commitment,
		ProposedFee:    proposedFee,
		Parameters:     parameters,
		asset:          p.asset,
		market:         p.market,
		party:          p.party,
		collateral:     p.collateral,
		position:       p.position,
		priceFactor:    p.priceFactor,
		positionFactor: p.positionFactor,
		status:         types.AMMPoolStatusActive,
		sqrt:           p.sqrt,
		oneTick:        p.oneTick,
	}
	if err := updated.setCurves(rf, sf, linearSlippage); err != nil {
		return nil, err
	}
	return updated, nil
}

// emptyCurve creates the curve details that represent no liquidity.
func emptyCurve(
	base *num.Uint,
) *curve {
	return &curve{
		l:     num.DecimalZero(),
		pv:    num.DecimalZero(),
		low:   base.Clone(),
		high:  base.Clone(),
		empty: true,
	}
}

// generateCurve creates the curve details and calculates its virtual liquidity.
func generateCurve(
	sqrt sqrtFn,
	commitment,
	low, high *num.Uint,
	riskFactor,
	marginFactor,
	linearSlippage num.Decimal,
	leverageAtBound *num.Decimal,
	positionFactor num.Decimal,
	isLower bool,
) *curve {
	// rf = 1 / ( mf * ( risk-factor + slippage ) )
	rf := num.DecimalOne().Div(marginFactor.Mul(riskFactor.Add(linearSlippage)))
	if leverageAtBound != nil {
		// rf = min(rf, leverage)
		rf = num.MinD(rf, *leverageAtBound)
	}

	// we now need to calculate the virtual-liquidity L of the curve from the
	// input parameters: leverage (rf), lower bound price (pl), upper bound price (pu)
	// we first calculate the unit-virtual-liquidity:
	// Lu = sqrt(pu) * sqrt(pl) / sqrt(pu) - sqrt(pl)

	// sqrt(high) * sqrt(low)
	term1 := sqrt(high).Mul(sqrt(low))

	// sqrt(high) - sqrt(low)
	term2 := sqrt(high).Sub(sqrt(low))
	lu := term1.Div(term2)

	// now we calculate average-entry price if we were to trade the entire curve
	// pa := lu * pu * (1 - (lu / lu + pu))

	// (1 - (lu / lu + pu))
	denom := num.DecimalOne().Sub(lu.Div(lu.Add(sqrt(high))))

	// lu * pu / denom
	pa := denom.Mul(lu).Mul(sqrt(high))

	// and now we calculate the theoretical position `pv` which is the total tradeable volume of the curve.
	var pv num.Decimal
	if isLower {
		// pv := rf * cc / ( pl(1 - rf) + rf * pa )

		// pl * (1 - rf)
		denom := low.ToDecimal().Mul(num.DecimalOne().Sub(rf))

		// ( pl(1 - rf) + rf * pa )
		denom = denom.Add(pa.Mul(rf))

		// pv := rf * cc / ( pl(1 - rf) + rf * pa )
		pv = commitment.ToDecimal().Mul(rf).Div(denom)
	} else {
		// pv := rf * cc / ( pu(1 + rf) - rf * pa )

		// pu * (1 + rf)
		denom := high.ToDecimal().Mul(num.DecimalOne().Add(rf))

		// ( pu(1 + rf) - rf * pa )
		denom = denom.Sub(pa.Mul(rf))

		// pv := rf * cc / ( pu(1 + rf) - rf * pa )
		pv = commitment.ToDecimal().Mul(rf).Div(denom).Abs()
	}

	// now we scale theoretical position by position factor so that is it feeds through into all subsequent equations
	pv = pv.Mul(positionFactor)

	// and finally calculate L = pv * Lu
	return &curve{
		l:    pv.Mul(lu),
		low:  low,
		high: high,
		pv:   pv,
	}
}

func (p *Pool) setCurves(
	rfs *types.RiskFactor,
	sfs *types.ScalingFactors,
	linearSlippage num.Decimal,
) error {
	// convert the bounds into asset precision
	base, _ := num.UintFromDecimal(p.Parameters.Base.ToDecimal().Mul(p.priceFactor))
	p.lower = emptyCurve(base)
	p.upper = emptyCurve(base)

	if p.Parameters.LowerBound != nil {
		lowerBound, _ := num.UintFromDecimal(p.Parameters.LowerBound.ToDecimal().Mul(p.priceFactor))
		p.lower = generateCurve(
			p.sqrt,
			p.Commitment.Clone(),
			lowerBound,
			base,
			rfs.Long,
			sfs.InitialMargin,
			linearSlippage,
			p.Parameters.LeverageAtLowerBound,
			p.positionFactor,
			true,
		)

		highPriceMinusOne := num.UintZero().Sub(p.lower.high, p.oneTick)
		// verify that the lower curve maintains sufficient volume from highPrice - 1 to the end of the curve.
		if p.lower.volumeBetweenPrices(p.sqrt, highPriceMinusOne, p.lower.high) < 1 {
			return fmt.Errorf("insufficient commitment - less than one volume at price levels on lower curve")
		}
	}

	if p.Parameters.UpperBound != nil {
		upperBound, _ := num.UintFromDecimal(p.Parameters.UpperBound.ToDecimal().Mul(p.priceFactor))
		p.upper = generateCurve(
			p.sqrt,
			p.Commitment.Clone(),
			base.Clone(),
			upperBound,
			rfs.Short,
			sfs.InitialMargin,
			linearSlippage,
			p.Parameters.LeverageAtUpperBound,
			p.positionFactor,
			false,
		)

		highPriceMinusOne := num.UintZero().Sub(p.upper.high, p.oneTick)
		// verify that the upper curve maintains sufficient volume from highPrice - 1 to the end of the curve.
		if p.upper.volumeBetweenPrices(p.sqrt, highPriceMinusOne, p.upper.high) < 1 {
			return fmt.Errorf("insufficient commitment - less than one volume at price levels on upper curve")
		}
	}

	return nil
}

// impliedPosition returns the position of the pool if its fair-price were the given price. `l` is
// the virtual liquidity of the pool, and `sqrtPrice` and `sqrtHigh` are, the square-roots of the
// price to calculate the position for, and higher boundary of the curve.
func impliedPosition(sqrtPrice, sqrtHigh num.Decimal, l num.Decimal) *num.Uint {
	// L * (sqrt(high) - sqrt(price))
	numer := sqrtHigh.Sub(sqrtPrice).Mul(l)

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

			volume := cu.volumeBetweenPrices(p.sqrt, price, next)
			if side == types.SideBuy && next.GT(fairPrice) {
				// now switch to sells, we're over the fair-price now
				side = types.SideSell
			}

			order := &types.Order{
				Size:  volume,
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

// PriceForVolume returns the price the AMM is willing to trade at to match with the given volume.
func (p *Pool) PriceForVolume(volume uint64, side types.Side) *num.Uint {
	x, y := p.virtualBalances(p.getPosition(), p.fairPrice(), side)

	// dy = x*y / (x - dx) - y
	// where y and x are the balances on either side of the pool, and dx is the change in volume
	// then the trade price is dy/dx
	dx := num.DecimalFromInt64(int64(volume))
	dy := x.Mul(y).Div(x.Sub(dx)).Sub(y)

	// dy / dx
	price, overflow := num.UintFromDecimal(dy.Div(dx))
	if overflow {
		panic("calculated negative price")
	}
	return price
}

// TradableVolumeInRange returns the volume the pool is willing to provide between the two given price levels for side of a given order
// that is trading with the pool. If `nil` is provided for either price then we take the full volume in that direction.
func (p *Pool) TradableVolumeInRange(side types.Side, price1 *num.Uint, price2 *num.Uint) uint64 {
	if !p.canTrade(side) {
		return 0
	}
	pos := p.getPosition()
	st, nd := price1, price2

	if price1 == nil {
		st = p.lower.low
	}

	if price2 == nil {
		nd = p.upper.high
	}

	if st.EQ(nd) {
		return 0
	}

	if st.GT(nd) {
		st, nd = nd, st
	}

	var other *curve
	var volume uint64
	// get the curve based on the pool's current position, if the position is zero we take the curve the trade will put us in
	// e.g trading with an incoming buy order will make the pool short, so we take the upper curve.
	if pos < 0 || (pos == 0 && side == types.SideBuy) {
		volume = p.upper.volumeBetweenPrices(p.sqrt, st, nd)
		other = p.lower
	} else {
		volume = p.lower.volumeBetweenPrices(p.sqrt, st, nd)
		other = p.upper
	}

	if p.closing() {
		return num.MinV(volume, uint64(num.AbsV(pos)))
	}

	// if the position is non-zero, the incoming order could push us across to the other curve
	// so we need to check for volume there too
	if pos != 0 {
		volume += other.volumeBetweenPrices(p.sqrt, st, nd)
	}
	return volume
}

// getBalance returns the total balance of the pool i.e it's general account + it's margin account.
func (p *Pool) getBalance() *num.Uint {
	general, err := p.collateral.GetPartyGeneralAccount(p.SubAccount, p.asset)
	if err != nil {
		panic("general account not created")
	}

	margin, err := p.collateral.GetPartyMarginAccount(p.market, p.SubAccount, p.asset)
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
		size: 0,
	}

	if pos := p.position.GetPositionsByParty(p.SubAccount); len(pos) != 0 {
		p.eph.size = pos[0].Size()
	}
}

// updateEphemeralPosition sets the pools transient position given a generated order.
func (p *Pool) updateEphemeralPosition(order *types.Order) {
	if order.Side == types.SideSell {
		p.eph.size -= int64(order.Size)
		return
	}
	p.eph.size += int64(order.Size)
}

// clearEphemeralPosition signifies that the matching process has finished
// and the pool can continue to read it's position from the positions engine.
func (p *Pool) clearEphemeralPosition() {
	p.eph = nil
}

// getPosition gets the pools current position an average-entry price.
func (p *Pool) getPosition() int64 {
	if p.eph != nil {
		return p.eph.size
	}

	if pos := p.position.GetPositionsByParty(p.SubAccount); len(pos) != 0 {
		return pos[0].Size()
	}
	return 0
}

// fairPrice returns the fair price of the pool given its current position.

// sqrt(pf) = sqrt(pu) / (1 + pv * sqrt(pu) * 1/L )
// where pv is the virtual-position
// pv = pos,  when the pool is long
// pv = pos + Pv, when pool is short
//
// this transformation is needed since for each curve its virtual position is 0 at the lower bound which maps to the Vega position when the pool is
// long, but when the pool is short Vega position == 0 at the upper bounds and -ve at the lower.
func (p *Pool) fairPrice() *num.Uint {
	pos := p.getPosition()
	if pos == 0 {
		// if no position fair price is base price
		return p.lower.high.Clone()
	}

	cu := p.lower
	pv := num.DecimalFromInt64(pos)
	if pos < 0 {
		cu = p.upper
		// pos + pv
		pv = cu.pv.Add(pv)
	}

	l := cu.l

	// pv * sqrt(pu) * (1/L) + 1
	denom := pv.Mul(p.sqrt(cu.high)).Div(l).Add(num.DecimalOne())

	// sqrt(fp) = sqrt(pu) / denom
	sqrtPf := p.sqrt(cu.high).Div(denom)

	// fair-price = sqrt(fp) * sqrt(fp)
	fp := sqrtPf.Mul(sqrtPf)

	// we want to round such that the price is further away from the base. This is so that once
	// a pool's position is at its boundary we do not report volume that doesn't exist. For example
	// say a pool's upper boundary is 1000 and for it to be at that boundary its position needs to
	// be 10.5. The closest we can get is 10 but then we'd report a fair-price of 999.78. If
	// we use 999 we'd be implying volume between 999 and 1000 which we don't want to trade.
	if pos < 0 {
		fp = fp.Ceil()
	}

	fairPrice, _ := num.UintFromDecimal(fp)
	return fairPrice
}

// virtualBalancesShort returns the pools x, y balances when the pool has a negative position
//
// x = P + Pv + L / sqrt(pl)
// y = L * sqrt(fair-price).
func (p *Pool) virtualBalancesShort(pos int64, fp *num.Uint) (num.Decimal, num.Decimal) {
	cu := p.upper
	if cu.empty {
		panic("should not be calculating balances on empty-curve side")
	}

	// lets start with x

	// P
	term1x := num.DecimalFromInt64(pos)

	// Pv
	term2x := cu.pv

	// L / sqrt(pl)
	term3x := cu.l.Div(p.sqrt(cu.high))

	// x = P + (cc * rf / pu) + (L / sqrt(pl))
	x := term2x.Add(term3x).Add(term1x)

	// now lets get y

	// y = L * sqrt(fair-price)
	y := cu.l.Mul(p.sqrt(fp))
	return x, y
}

// virtualBalancesLong returns the pools x, y balances when the pool has a positive position
//
// x = P + (L / sqrt(pu))
// y = L * sqrt(fair-price).
func (p *Pool) virtualBalancesLong(pos int64, fp *num.Uint) (num.Decimal, num.Decimal) {
	cu := p.lower
	if cu.empty {
		panic("should not be calculating balances on empty-curve side")
	}

	// lets start with x

	// P
	term1x := num.DecimalFromInt64(pos)

	// L / sqrt(pu)
	term2x := cu.l.Div(p.sqrt(cu.high))

	// x = P + (L / sqrt(pu))
	x := term1x.Add(term2x)

	// now lets move to y

	// y = L * sqrt(fair-price)
	y := cu.l.Mul(p.sqrt(fp))
	return x, y
}

// virtualBalances returns the pools x, y values where x is the balance in contracts and y is the balance in asset.
func (p *Pool) virtualBalances(pos int64, fp *num.Uint, side types.Side) (num.Decimal, num.Decimal) {
	switch {
	case pos < 0, pos == 0 && side == types.SideBuy:
		// zero position but incoming is buy which will make pool short
		return p.virtualBalancesShort(pos, fp)
	case pos > 0, pos == 0 && side == types.SideSell:
		// zero position but incoming is sell which will make pool long
		return p.virtualBalancesLong(pos, fp)
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

func (p *Pool) LiquidityFee() num.Decimal {
	return p.ProposedFee
}

func (p *Pool) CommitmentAmount() *num.Uint {
	return p.Commitment.Clone()
}

func (p *Pool) Owner() string {
	return p.party
}

func (p *Pool) closing() bool {
	return p.status == types.AMMPoolStatusReduceOnly
}

func (p *Pool) canTrade(side types.Side) bool {
	if !p.closing() {
		return true
	}

	pos := p.getPosition()
	// pool is long incoming order is a buy and will make it shorter, its ok
	if pos > 0 && side == types.SideBuy {
		return true
	}
	if pos < 0 && side == types.SideSell {
		return true
	}
	return false
}
