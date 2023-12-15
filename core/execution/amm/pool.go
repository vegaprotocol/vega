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
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

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

	asset       string
	market      string
	party       string
	collateral  Collateral
	position    Position
	priceFactor *num.Uint

	// sqrt function to use.
	sqrt sqrtFn

	// the two curves joined at base-price used to determine price and volume in the pool
	// lower is used when the pool is long.
	lower *curve
	upper *curve
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
) *Pool {
	pool := &Pool{
		ID:          id,
		SubAccount:  subAccount,
		Commitment:  submit.CommitmentAmount,
		Parameters:  submit.Parameters,
		market:      submit.MarketID,
		party:       submit.Party,
		asset:       asset,
		sqrt:        sqrt,
		collateral:  collateral,
		position:    position,
		priceFactor: priceFactor,
	}
	pool.setCurves(rf, sf, linearSlippage)
	return pool
}

func NewPoolFromProto(
	sqrt sqrtFn,
	collateral Collateral,
	position Position,
	state *snapshotpb.PoolMapEntry_Pool,
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
) *curve {
	// rf = 1 / ( mf * ( risk-factor + slippage ) )
	rf := num.DecimalOne().Div(marginFactor.Mul(riskFactor.Add(linearSlippage)))
	if marginRatio != nil {
		// rf = min(rf, 1/margin-ratio)
		rf = num.MinD(rf, num.DecimalOne().Div(*marginRatio))
	}

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

// VolumeBetweenPrices returns the volume the pool is willing to provide between the two given price levels for side of a given order
// being placed by the pool.
func (p *Pool) VolumeBetweenPrices(side types.Side, price1 *num.Uint, price2 *num.Uint) uint64 {
	var pos int64
	if pp := p.position.GetPositionsByParty(p.SubAccount); len(pp) > 0 {
		pos = pp[0].Size()
	}

	st, nd := price1, price2
	if st.EQ(nd) {
		return 0
	}

	if st.GT(nd) {
		st, nd = nd, st
	}

	// get the curve based on the pool's current position, if the position is zero we take the curve the trade will put us in
	// e.g trading with a sell order will make the pool short, so we take the upper curve.
	var cu *curve
	if pos < 0 || (pos == 0 && side == types.SideSell) {
		cu = p.upper
	} else {
		cu = p.lower
	}

	// there is no volume outside of the bounds for the curve so we snap st, nd to the boundaries
	st = num.Max(st, cu.low)
	nd = num.Min(nd, cu.high)

	// abs(P(st) - P(nd))
	volume, _ := num.UintZero().Delta(
		impliedPosition(p.sqrt(st), p.sqrt(cu.high), cu.l),
		impliedPosition(p.sqrt(nd), p.sqrt(cu.high), cu.l),
	)
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

// getPosition gets the pools current position an average-entry price.
func (p *Pool) getPosition() (int64, *num.Uint) {
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
	// balance := p.getBalance()

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

// virtualBalances returns the pools x, y values where x is the balance in contracts and y is the balance in asset.
func (p *Pool) fairPrice() *num.Uint {
	fairPrice := num.UintZero()
	pos, ae := p.getPosition()

	switch {
	case pos == 0:
		fairPrice = p.lower.high.Clone()
	case pos < 0:
		x, y := p.virtualBalancesShort(pos, ae)
		fairPrice, _ = num.UintFromDecimal(y.Div(x))
	case pos > 0:
		x, y := p.virtualBalancesLong(pos, ae)
		fairPrice, _ = num.UintFromDecimal(y.Div(x))
	}

	return fairPrice
}

// TradePrice returns the price that the pool is willing to trade for the given order and its volume.
func (p *Pool) TradePrice(order *types.Order) *num.Uint {
	fairPrice := p.fairPrice()
	switch {
	case order == nil:
		// special case where we've been asked for a fair price
		return fairPrice
	case order.Side == types.SideBuy:
		// incoming is a buy, so we +1 to the fair price
		return fairPrice.AddSum(num.UintOne())
	case order.Side == types.SideSell:
		// incoming is a sell so we - 1 the fair price
		return fairPrice.Sub(fairPrice, num.UintOne())
	default:
		panic("should never reach here")
	}
}
