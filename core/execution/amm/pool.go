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
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/core/idgeneration"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/logging"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

// ephemeralPosition keeps track of the pools position as if its generated orders had traded.
type ephemeralPosition struct {
	size int64
}

type curve struct {
	l       num.Decimal // virtual liquidity
	high    *num.Uint   // high price value, upper bound if upper curve, base price is lower curve
	low     *num.Uint   // low price value, base price if upper curve, lower bound if lower curve
	empty   bool        // if true the curve is of zero length and represents no liquidity on this side of the amm
	isLower bool        // whether the curve is for the lower curve or the upper curve

	// the theoretical position of the curve at its lower boundary
	// note that this equals Vega's position at the boundary only in the lower curve, since Vega position == curve-position
	// in the upper curve Vega's position == 0 => position of `pv`` in curve-position, Vega's position pv => 0 in curve-position
	pv num.Decimal

	lDivSqrtPu num.Decimal
	sqrtHigh   num.Decimal
}

// positionAtPrice returns the position of the AMM if its fair-price were the given price. This
// will be signed for long/short as usual.
func (c *curve) positionAtPrice(sqrt sqrtFn, price *num.Uint) int64 {
	pos := impliedPosition(sqrt(price), c.sqrtHigh, c.l)
	if c.isLower {
		return pos.IntPart()
	}

	// if we are in the upper curve the position of 0 in "curve-space" is -cu.pv in Vega position
	// so we need to flip the interval
	return -c.pv.Sub(pos).IntPart()
}

// singleVolumePrice returns the price that is 1 volume away from the given price in the given direction.
// If the AMM's commitment is low this may be more than one-tick away from `p`.
func (c *curve) singleVolumePrice(sqrt sqrtFn, p *num.Uint, side types.Side) *num.Uint {
	if c.empty {
		panic("should not be calculating single-volume step on empty curve")
	}

	// for best buy:  (L * sqrt(pu) / (L + sqrt(pu)))^2
	// for best sell: (L * sqrt(pu) / (L - sqrt(pu)))^2
	var denom num.Decimal
	if side == types.SideBuy {
		denom = c.l.Add(sqrt(p))
	} else {
		denom = c.l.Sub(sqrt(p))
	}

	np := c.l.Mul(sqrt(p)).Div(denom)
	np = np.Mul(np)

	if side == types.SideSell {
		// have to make sure we round away `p`
		np = np.Ceil()
	}

	adj, _ := num.UintFromDecimal(np)
	return adj
}

// singleVolumeDelta returns the price interval between p and the price that represents 1 volume movement.
func (c *curve) singleVolumeDelta(sqrt sqrtFn, p *num.Uint, side types.Side) *num.Uint {
	adj := c.singleVolumePrice(sqrt, p, side)
	delta, _ := num.UintZero().Delta(p, adj)
	return delta
}

// check will return an error is the curve contains too many price-levels where there is 0 volume.
func (c *curve) check(sqrt sqrtFn, oneTick *num.Uint, allowedEmptyLevels uint64) error {
	if c.empty {
		return nil
	}

	if c.pv.LessThan(num.DecimalOne()) {
		return ErrCommitmentTooLow
	}

	// curve is valid if
	// n * oneTick > pu - (L * sqrt(pu) / (L + sqrt(pu)))^2
	adj := c.singleVolumePrice(sqrt, c.high, types.SideBuy)
	delta := num.UintZero().Sub(c.high, adj)

	// the plus one is because if allowable empty levels is 0, then the biggest delta allowed is 1
	maxDelta := num.UintZero().Mul(oneTick, num.NewUint(allowedEmptyLevels+1))

	// now this price delta must be less that the given maximum
	if delta.GT(maxDelta) {
		return ErrCommitmentTooLow
	}
	return nil
}

type Pool struct {
	log         *logging.Logger
	ID          string
	AMMParty    string
	Commitment  *num.Uint
	ProposedFee num.Decimal
	Parameters  *types.ConcentratedLiquidityParameters

	asset          string
	market         string
	owner          string
	collateral     Collateral
	position       Position
	priceFactor    num.Decimal
	positionFactor num.Decimal

	spread                    num.Decimal
	SlippageTolerance         num.Decimal
	MinimumPriceChangeTrigger num.Decimal

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

	maxCalculationLevels *num.Uint // maximum number of price levels the AMM will be expanded into
	oneTick              *num.Uint // one price tick

	cache *poolCache
}

func NewPool(
	log *logging.Logger,
	id,
	ammParty,
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
	maxCalculationLevels *num.Uint,
	allowedEmptyAMMLevels uint64,
) (*Pool, error) {
	oneTick, _ := num.UintFromDecimal(priceFactor)
	pool := &Pool{
		log:                       log,
		ID:                        id,
		AMMParty:                  ammParty,
		Commitment:                submit.CommitmentAmount,
		ProposedFee:               submit.ProposedFee,
		Parameters:                submit.Parameters,
		market:                    submit.MarketID,
		owner:                     submit.Party,
		asset:                     asset,
		sqrt:                      sqrt,
		collateral:                collateral,
		position:                  position,
		priceFactor:               priceFactor,
		positionFactor:            positionFactor,
		oneTick:                   num.Max(num.UintOne(), oneTick),
		status:                    types.AMMPoolStatusActive,
		maxCalculationLevels:      maxCalculationLevels,
		cache:                     NewPoolCache(),
		spread:                    submit.Spread,
		SlippageTolerance:         submit.SlippageTolerance,
		MinimumPriceChangeTrigger: submit.MinimumPriceChangeTrigger,
	}

	if submit.Parameters.DataSourceID != nil {
		pool.status = types.AMMPoolStatusPending
		pool.lower = emptyCurve(num.UintZero(), true)
		pool.upper = emptyCurve(num.UintZero(), false)
		return pool, nil
	}

	err := pool.setCurves(rf, sf, linearSlippage, allowedEmptyAMMLevels)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

func NewPoolFromProto(
	log *logging.Logger,
	sqrt sqrtFn,
	collateral Collateral,
	position Position,
	state *snapshotpb.PoolMapEntry_Pool,
	party string,
	priceFactor num.Decimal,
	positionFactor num.Decimal,
) (*Pool, error) {
	oneTick, _ := num.UintFromDecimal(priceFactor)

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

	var lower, upper *num.Uint
	if state.Parameters.LowerBound != nil {
		lower, overflow = num.UintFromString(*state.Parameters.LowerBound, 10)
		if overflow {
			return nil, fmt.Errorf("failed to convert string to Uint: %s", *state.Parameters.LowerBound)
		}
	}

	if state.Parameters.UpperBound != nil {
		upper, overflow = num.UintFromString(*state.Parameters.UpperBound, 10)
		if overflow {
			return nil, fmt.Errorf("failed to convert string to Uint: %s", *state.Parameters.UpperBound)
		}
	}

	upperCu, err := NewCurveFromProto(state.Upper)
	if err != nil {
		return nil, err
	}

	lowerCu, err := NewCurveFromProto(state.Lower)
	lowerCu.isLower = true
	if err != nil {
		return nil, err
	}

	proposedFee, err := num.DecimalFromString(state.ProposedFee)
	if err != nil {
		return nil, err
	}

	var slippageTolerance num.Decimal
	if state.SlippageTolerance != "" {
		slippageTolerance, err = num.DecimalFromString(state.SlippageTolerance)
		if err != nil {
			return nil, err
		}
	}

	minimumPriceChangeTrigger := num.DecimalZero()
	if state.MinimumPriceChangeTrigger != "" {
		minimumPriceChangeTrigger, err = num.DecimalFromString(state.MinimumPriceChangeTrigger)
		if err != nil {
			return nil, err
		}
	}

	return &Pool{
		log:         log,
		ID:          state.Id,
		AMMParty:    state.AmmPartyId,
		Commitment:  num.MustUintFromString(state.Commitment, 10),
		ProposedFee: proposedFee,
		Parameters: &types.ConcentratedLiquidityParameters{
			Base:                 base,
			LowerBound:           lower,
			UpperBound:           upper,
			LeverageAtLowerBound: lowerLeverage,
			LeverageAtUpperBound: upperLeverage,
			DataSourceID:         state.Parameters.DataSourceId,
		},
		owner:                     party,
		market:                    state.Market,
		asset:                     state.Asset,
		sqrt:                      sqrt,
		collateral:                collateral,
		position:                  position,
		lower:                     lowerCu,
		upper:                     upperCu,
		priceFactor:               priceFactor,
		positionFactor:            positionFactor,
		oneTick:                   num.Max(num.UintOne(), oneTick),
		status:                    state.Status,
		cache:                     NewPoolCache(),
		SlippageTolerance:         slippageTolerance,
		MinimumPriceChangeTrigger: minimumPriceChangeTrigger,
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

	var sqrtHigh, lDivSqrtPu num.Decimal
	if !c.Empty {
		sqrtHigh = num.UintOne().Sqrt(high)
		lDivSqrtPu = l.Div(sqrtHigh)
	}

	return &curve{
		l:          l,
		high:       high,
		low:        low,
		empty:      c.Empty,
		pv:         pv,
		sqrtHigh:   sqrtHigh,
		lDivSqrtPu: lDivSqrtPu,
	}, nil
}

func (p *Pool) IntoProto() *snapshotpb.PoolMapEntry_Pool {
	return &snapshotpb.PoolMapEntry_Pool{
		Id:          p.ID,
		AmmPartyId:  p.AMMParty,
		Commitment:  p.Commitment.String(),
		ProposedFee: p.ProposedFee.String(),
		Parameters:  p.Parameters.ToProtoEvent(),
		Market:      p.market,
		Asset:       p.asset,
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
			Pv:    p.upper.pv.String(),
		},
		Status:                    p.status,
		SlippageTolerance:         p.SlippageTolerance.String(),
		MinimumPriceChangeTrigger: p.MinimumPriceChangeTrigger.String(),
	}
}

// checkPosition will return false if its position exists outside of the curve boundaries and so the AMM
// is invalid.
func (p *Pool) checkPosition() bool {
	pos := p.getPosition()

	if pos > p.lower.pv.IntPart() {
		return false
	}

	if -pos > p.upper.pv.IntPart() {
		return false
	}

	return true
}

// Update returns a copy of the give pool but with its curves and parameters update as specified by `amend`.
func (p *Pool) Update(
	amend *types.AmendAMM,
	rf *types.RiskFactor,
	sf *types.ScalingFactors,
	linearSlippage num.Decimal,
	allowedEmptyAMMLevels uint64,
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

	// if an AMM is amended so that it cannot be long (i.e it has no lower curve) but the existing AMM
	// is already long then we cannot make the change since its fair-price will be undefined.
	if parameters.LowerBound == nil && p.getPosition() > 0 {
		return nil, errors.New("cannot remove lower bound when AMM is long")
	}

	if parameters.UpperBound == nil && p.getPosition() < 0 {
		return nil, errors.New("cannot remove upper bound when AMM is short")
	}

	updated := &Pool{
		log:                       p.log,
		ID:                        p.ID,
		AMMParty:                  p.AMMParty,
		Commitment:                commitment,
		ProposedFee:               proposedFee,
		Parameters:                parameters,
		asset:                     p.asset,
		market:                    p.market,
		owner:                     p.owner,
		collateral:                p.collateral,
		position:                  p.position,
		priceFactor:               p.priceFactor,
		positionFactor:            p.positionFactor,
		status:                    types.AMMPoolStatusActive,
		sqrt:                      p.sqrt,
		oneTick:                   p.oneTick,
		maxCalculationLevels:      p.maxCalculationLevels,
		cache:                     NewPoolCache(),
		SlippageTolerance:         amend.SlippageTolerance,
		MinimumPriceChangeTrigger: amend.MinimumPriceChangeTrigger,
		spread:                    amend.Spread,
	}

	// data source has changed, if the old base price is within bounds we'll keep it until the update comes in
	// otherwise we'll kick it into pending
	if ptr.UnBox(parameters.DataSourceID) != ptr.UnBox(p.Parameters.DataSourceID) {
		base := p.lower.high
		outside := p.IsPending()

		if parameters.UpperBound != nil {
			bound, _ := num.UintFromDecimal(parameters.UpperBound.ToDecimal().Mul(p.priceFactor))
			outside = outside || base.GTE(bound)
		}

		if parameters.LowerBound != nil {
			bound, _ := num.UintFromDecimal(parameters.LowerBound.ToDecimal().Mul(p.priceFactor))
			outside = outside || base.LTE(bound)
		}

		if outside {
			updated.status = types.AMMPoolStatusPending
			updated.lower = emptyCurve(num.UintZero(), true)
			updated.upper = emptyCurve(num.UintZero(), false)
			return updated, nil
		}

		// inherit the old base price
		parameters.Base = p.Parameters.Base.Clone()
	}

	if err := updated.setCurves(rf, sf, linearSlippage, allowedEmptyAMMLevels); err != nil {
		return nil, err
	}

	if !updated.checkPosition() {
		return nil, errors.New("AMM's current position is outside of amended bounds - reduce position first")
	}

	return updated, nil
}

// emptyCurve creates the curve details that represent no liquidity.
func emptyCurve(
	base *num.Uint,
	isLower bool,
) *curve {
	return &curve{
		l:       num.DecimalZero(),
		pv:      num.DecimalZero(),
		low:     base.Clone(),
		high:    base.Clone(),
		empty:   true,
		isLower: isLower,
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
	l := pv.Mul(lu)

	sqrtHigh := sqrt(high)
	lDivSqrtPu := l.Div(sqrtHigh)

	// and finally calculate L = pv * Lu
	return &curve{
		l:          l,
		low:        low,
		high:       high,
		pv:         pv,
		isLower:    isLower,
		lDivSqrtPu: lDivSqrtPu,
		sqrtHigh:   sqrtHigh,
	}
}

func (p *Pool) setCurves(
	rfs *types.RiskFactor,
	sfs *types.ScalingFactors,
	linearSlippage num.Decimal,
	allowedEmptyAMMLevels uint64,
) error {
	// convert the bounds into asset precision
	base, _ := num.UintFromDecimal(p.Parameters.Base.ToDecimal().Mul(p.priceFactor))
	p.lower = emptyCurve(base, true)
	p.upper = emptyCurve(base, false)

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

		if err := p.lower.check(p.sqrt, p.oneTick.Clone(), allowedEmptyAMMLevels); err != nil {
			return err
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

		// lets find an interval that represents one volume, it might be a sparse curve
		if err := p.upper.check(p.sqrt, p.oneTick.Clone(), allowedEmptyAMMLevels); err != nil {
			return err
		}
	}

	return nil
}

// impliedPosition returns the position of the pool if its fair-price were the given price. `l` is
// the virtual liquidity of the pool, and `sqrtPrice` and `sqrtHigh` are, the square-roots of the
// price to calculate the position for, and higher boundary of the curve.
func impliedPosition(sqrtPrice, sqrtHigh num.Decimal, l num.Decimal) num.Decimal {
	// L * (sqrt(high) - sqrt(price))
	numer := sqrtHigh.Sub(sqrtPrice).Mul(l)

	// sqrt(high) * sqrt(price)
	denom := sqrtHigh.Mul(sqrtPrice)

	// L * (sqrt(high) - sqrt(price)) / sqrt(high) * sqrt(price)
	return numer.Div(denom)
}

// PriceForVolume returns the price the AMM is willing to trade at to match with the given volume of an incoming order.
func (p *Pool) PriceForVolume(volume uint64, side types.Side) *num.Uint {
	return p.priceForVolumeAtPosition(
		volume,
		side,
		p.getPosition(),
		p.FairPrice(),
	)
}

// priceForVolumeAtPosition returns the price the AMM is willing to trade at to match with the given volume if its position and fair-price
// are as given.
func (p *Pool) priceForVolumeAtPosition(volume uint64, side types.Side, pos int64, fp *num.Uint) *num.Uint {
	if volume == 0 {
		panic("cannot calculate price for zero volume trade")
	}

	x, y := p.virtualBalances(pos, fp, side)

	// dy = x*y / (x - dx) - y
	// where y and x are the balances on either side of the pool, and dx is the change in volume
	// then the trade price is dy/dx
	dx := num.DecimalFromInt64(int64(volume))
	if side == types.SideSell {
		// if incoming order is a sell, the AMM is buying so reducing cash balance so dx is negative
		dx = dx.Neg()
	}

	dy := x.Mul(y).Div(x.Sub(dx)).Sub(y)

	// dy / dx
	price, overflow := num.UintFromDecimal(dy.Div(dx).Abs())
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

	// map the given st/nd prices into positions, then the difference is the volume
	asPosition := func(price *num.Uint) int64 {
		switch {
		case price.GT(p.lower.high):
			// in upper curve
			if !p.upper.empty {
				return p.upper.positionAtPrice(p.sqrt, num.Min(p.upper.high, price))
			}
		case price.LT(p.lower.high):
			// in lower curve
			if !p.lower.empty {
				return p.lower.positionAtPrice(p.sqrt, num.Max(p.lower.low, price))
			}
		}
		return 0
	}

	stP := asPosition(st)
	ndP := asPosition(nd)

	if side == types.SideSell {
		// want all buy volume so everything below fair price, where the AMM is long
		if pos > stP {
			return 0
		}
		ndP = num.MaxV(pos, ndP)
	}

	if side == types.SideBuy {
		// want all sell volume so everything above fair price, where the AMM is short
		if pos < ndP {
			return 0
		}
		stP = num.MinV(pos, stP)
	}

	if !p.closing() {
		return uint64(stP - ndP)
	}

	if pos > 0 {
		// if closing and long, we have no volume at short prices, so cap range to > 0
		stP = num.MaxV(0, stP)
		ndP = num.MaxV(0, ndP)
	}

	if pos < 0 {
		// if closing and short, we have no volume at long prices, so cap range to < 0
		stP = num.MinV(0, stP)
		ndP = num.MinV(0, ndP)
	}
	return num.MinV(uint64(stP-ndP), uint64(num.AbsV(pos)))
}

// TrableVolumeForPrice returns the volume available between the AMM's fair-price and the given
// price and side of an incoming order. It is a special case of TradableVolumeInRange with
// the benefit of accurately using the AMM's position instead of having to calculate the hop
// from fair-price -> position.
func (p *Pool) TradableVolumeForPrice(side types.Side, price *num.Uint) uint64 {
	if side == types.SideSell {
		return p.TradableVolumeInRange(side, price, nil)
	}
	return p.TradableVolumeInRange(side, nil, price)
}

// getBalance returns the total balance of the pool i.e it's general account + it's margin account.
func (p *Pool) getBalance() *num.Uint {
	general, err := p.collateral.GetPartyGeneralAccount(p.AMMParty, p.asset)
	if err != nil {
		panic("general account not created")
	}

	margin, err := p.collateral.GetPartyMarginAccount(p.market, p.AMMParty, p.asset)
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

	if pos := p.position.GetPositionsByParty(p.AMMParty); len(pos) != 0 {
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

	if pos := p.position.GetPositionsByParty(p.AMMParty); len(pos) != 0 {
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
func (p *Pool) FairPrice() *num.Uint {
	pos := p.getPosition()
	if pos == 0 {
		// if no position fair price is base price
		return p.lower.high.Clone()
	}

	if fp, ok := p.cache.getFairPrice(pos); ok {
		return fp.Clone()
	}

	cu := p.lower
	pv := num.DecimalFromInt64(pos)
	if pos < 0 {
		cu = p.upper
		// pos + pv
		pv = cu.pv.Add(pv)
	}

	if cu.empty {
		p.log.Panic("should not be calculating fair-price on empty-curve side",
			logging.Bool("lower", cu.isLower),
			logging.Int64("pos", pos),
			logging.String("amm-party", p.AMMParty),
		)
	}

	// pv * sqrt(pu) * (1/L) + 1
	denom := pv.Mul(cu.sqrtHigh).Div(cu.l).Add(num.DecimalOne())

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

	p.cache.setFairPrice(pos, fairPrice.Clone())

	return fairPrice
}

// virtualBalancesShort returns the pools x, y balances when the pool has a negative position
//
// x = P + Pv + L / sqrt(pu)
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

	// L / sqrt(pu)
	term3x := cu.lDivSqrtPu

	// x = P + (cc * rf / pu) + (L / sqrt(pu))
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
	term2x := cu.lDivSqrtPu

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

// BestPrice returns the AMM's quote price on the given side. If the AMM's position is fully at a boundary
// then there is no quote price on that side and false is returned.
func (p *Pool) BestPrice(side types.Side) (*num.Uint, bool) {
	if p.IsPending() {
		return nil, false
	}

	pos := p.getPosition()
	fairPrice := p.FairPrice()

	switch side {
	case types.SideSell:
		cu := p.lower
		if pos <= 0 {
			cu = p.upper
			// we're short, and want the sell quote price, if we're at the boundary there is not volume left
			if p.closing() || num.AbsV(pos) >= cu.pv.IntPart() {
				return nil, false
			}
		}

		np := cu.singleVolumePrice(p.sqrt, fairPrice, side)
		return num.Min(p.upper.high, num.Max(np, fairPrice.AddSum(p.oneTick))), true
	case types.SideBuy:
		cu := p.upper
		if pos >= 0 {
			cu = p.lower
			// we're long, and want the buy quote price, if we're at the boundary there is not volume left
			if p.closing() || pos >= cu.pv.IntPart() {
				return nil, false
			}
		}

		np := cu.singleVolumePrice(p.sqrt, fairPrice, side)
		return num.Max(p.lower.low, num.Min(np, num.UintZero().Sub(fairPrice, p.oneTick))), true
	default:
		panic("should never reach here")
	}
}

// BestPriceAndVolume returns the AMM's best price on a given side and the volume available to trade.
func (p *Pool) BestPriceAndVolume(side types.Side) (*num.Uint, uint64) {
	// check cache
	pos := p.getPosition()

	if p, v, ok := p.cache.getBestPrice(pos, side, p.status); ok {
		return p, v
	}

	price, ok := p.BestPrice(side)
	if !ok {
		return price, 0
	}

	// now calculate the volume
	fp := p.FairPrice()
	if side == types.SideBuy {
		priceTick := num.Max(p.lower.low, num.UintZero().Sub(fp, p.oneTick))

		if !price.GTE(priceTick) {
			p.cache.setBestPrice(pos, side, p.status, price, 1)
			return price, 1 // its low volume so 1 by construction
		}

		volume := p.TradableVolumeForPrice(types.SideSell, priceTick)
		p.cache.setBestPrice(pos, side, p.status, priceTick, volume)
		return priceTick, volume
	}

	priceTick := num.Min(p.upper.high, num.UintZero().Add(fp, p.oneTick))
	if !price.LTE(priceTick) {
		p.cache.setBestPrice(pos, side, p.status, price, 1)
		return price, 1 // its low volume so 1 by construction
	}

	volume := p.TradableVolumeForPrice(types.SideBuy, priceTick)
	p.cache.setBestPrice(pos, side, p.status, priceTick, volume)
	return priceTick, volume
}

func (p *Pool) LiquidityFee() num.Decimal {
	return p.ProposedFee
}

func (p *Pool) CommitmentAmount() *num.Uint {
	return p.Commitment.Clone()
}

func (p *Pool) Owner() string {
	return p.owner
}

func (p *Pool) closing() bool {
	return p.status == types.AMMPoolStatusReduceOnly
}

func (p *Pool) IsPending() bool {
	return p.status == types.AMMPoolStatusPending
}

func (p *Pool) canTrade(side types.Side) bool {
	if p.IsPending() {
		return false
	}

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

func (p *Pool) makeOrder(volume uint64, price *num.Uint, side types.Side, idgen *idgeneration.IDGenerator) *types.Order {
	order := &types.Order{
		MarketID:         p.market,
		Party:            p.AMMParty,
		Size:             volume,
		Remaining:        volume,
		Price:            price,
		Side:             side,
		TimeInForce:      types.OrderTimeInForceGTC,
		Type:             types.OrderTypeLimit,
		Status:           types.OrderStatusFilled,
		Reference:        "vamm-" + p.AMMParty,
		GeneratedOffbook: true,
	}
	order.OriginalPrice, _ = num.UintFromDecimal(order.Price.ToDecimal().Div(p.priceFactor))

	if idgen != nil {
		order.ID = idgen.NextID()
	}
	return order
}
