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

package service

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

var activeStates = []entities.AMMStatus{entities.AMMStatusActive, entities.AMMStatusReduceOnly}

// TODO make this configurable
const (
	// accurate expansion 5% either side of mid
	accurateExpansion = 0.01
	// step size in estimated region 10% of mid
	estimateStep = 0.1
	// the number of steps to take in the estimated region
	maxEstimatedSteps = 5
)

type amm struct {
	entities.AMMPool
	lower    *curve
	upper    *curve
	position int64 // signed position in Vega-space
}

type curve struct {
	pv      num.Decimal
	l       num.Decimal
	isLower bool
}

func (cu *curve) impliedPosition(price, high *num.Uint) num.Decimal {
	sqrtHigh := high.Sqrt(high)
	sqrtPrice := price.Sqrt(price)

	// L * (sqrt(high) - sqrt(price))
	numer := sqrtHigh.Sub(sqrtPrice).Mul(cu.l)

	// sqrt(high) * sqrt(price)
	denom := sqrtHigh.Mul(sqrtPrice)

	// L * (sqrt(high) - sqrt(price)) / sqrt(high) * sqrt(price)
	res := numer.Div(denom)

	if cu.isLower {
		return res
	}

	// if we are in the upper curve the position of 0 in "curve-space" is -cu.pv in Vega position
	// so we need to flip the interval
	return cu.pv.Sub(res).Neg()

}

func (m *MarketDepth) GetActiveAMMs(ctx context.Context) map[string][]entities.AMMPool {

	ammByMarket := map[string][]entities.AMMPool{}
	for _, state := range activeStates {

		// TODO page through AMMs
		amms, _, err := m.ammStore.ListByStatus(ctx, state, entities.DefaultCursorPagination(true))
		if err != nil {
			m.log.Warn("unable to query AMM's for market-depth",
				logging.Error(err),
			)
			continue
		}

		for _, amm := range amms {
			marketID := string(amm.MarketID)
			if _, ok := ammByMarket[marketID]; !ok {
				ammByMarket[marketID] = []entities.AMMPool{}
			}

			ammByMarket[marketID] = append(ammByMarket[marketID], amm)
		}
	}
	return ammByMarket
}

func (m *MarketDepth) ExpandAMMs(ctx context.Context) error {

	active := m.GetActiveAMMs(ctx)
	if len(active) == 0 {
		return nil
	}

	// expand all these AMM's from the midpoint
	for marketID, amms := range active {

		marketData, err := m.marketData.GetMarketDataByID(ctx, marketID)
		if err != nil {
			m.log.Warn("unable to get market-data for market",
				logging.String("market-id", marketID),
				logging.Error(err),
			)
			continue
		}

		reference := marketData.MidPrice
		if !marketData.IndicativePrice.IsZero() {
			reference = marketData.IndicativePrice
		}

		if reference.IsZero() {
			m.log.Warn("cannot calculate market-depth for AMM, no reference point available",
				logging.String("mid-price", marketData.MidPrice.String()),
				logging.String("indicative-price", marketData.IndicativePrice.String()),
			)
			continue
		}

		// TODO scale reference to Asset DP?
		for _, amm := range amms {
			m.ExpandAMM(amm, reference)
		}

	}

	return nil
}

func (m *MarketDepth) ExpandAMM(pool entities.AMMPool, reference num.Decimal) error {

	// get positions
	pos, err := m.positions.GetByMarketAndParty(context.Background(), string(pool.MarketID), string(pool.AmmPartyID))
	if err != nil {
		// TODO if not found positions is zero, anything else is real error
	}
	amm := &amm{
		AMMPool:  pool,
		position: pos.OpenVolume,
		lower: &curve{
			isLower: true,
			l:       pool.LowerVirtualLiquidity,
			pv:      pool.LowerTheoreticalPosition,
		},
		upper: &curve{
			l:  pool.UpperVirtualLiquidity,
			pv: pool.UpperTheoreticalPosition,
		},
	}

	// pack some curves

	// calculate accurate bounds
	//accLow, _ := reference.Mul(num.DecimalOne().Add(num.DecimalFromFloat(accurateExpansion))).Uint()
	accHigh, _ := num.UintFromDecimal(reference.Mul(num.DecimalOne().Add(num.DecimalFromFloat(accurateExpansion))))
	accLow, _ := num.UintFromDecimal(reference.Mul(num.DecimalOne().Sub(num.DecimalFromFloat(accurateExpansion))))

	eStep, _ := num.UintFromDecimal(reference.Mul(num.DecimalFromFloat(estimateStep)))

	eRange := num.UintZero().Mul(eStep, num.NewUint(maxEstimatedSteps))
	estLow := num.UintZero().Sub(accLow, eRange)
	estHigh := num.UintZero().Add(accHigh, eRange)

	// lets calculate all the prices we are going to get a level for
	levels := []*num.Uint{estLow.Clone()}

	fmt.Println("level regions", estLow, accLow, accHigh, estHigh)

	step := eStep.Clone()
	price := estLow.Clone()
	estimate := true
	for price.LTE(estHigh) {

		// if this price is in the AMM's range calculate the thing.

		next := num.UintZero().Add(price, step)
		levels = append(levels, next.Clone())

		// get which curve we're on
		m.getVolume(amm, price, next)

		// if we're entering accurate region, reduce step size
		if estimate && next.GTE(accLow) {
			step = num.UintOne()
			estimate = false
		}

		// if we've leaving accurate region increase step size
		if !estimate && next.GTE(accHigh) {
			step = eStep.Clone()
			estimate = true
		}
		price = next
	}

	// all we need is implied position at each price, then we're golden

	return nil
}

func (m *MarketDepth) getVolume(pool *amm, price1, price2 *num.Uint) (uint64, *num.Uint) {

	// cap the ranges
	// now get the range the AMM itself lives in

	// the AMM's position is its fair-price, we don't want to calculate the fair-price from the position

	base, _ := num.UintFromDecimal(pool.ParametersBase)
	pLow, _ := num.UintFromDecimal(pool.ParametersBase)
	if pool.ParametersLowerBound != nil {
		pLow, _ = num.UintFromDecimal(*pool.ParametersLowerBound)
	}

	pHigh, _ := num.UintFromDecimal(pool.ParametersBase)
	if pool.ParametersUpperBound != nil {
		pHigh, _ = num.UintFromDecimal(*pool.ParametersUpperBound)
	}

	// outside of range
	if pLow.GTE(price2) {
		fmt.Println("volume", price1, price2, 0, "outside low")
		return 0, nil
	}

	if pHigh.LTE(price1) {
		fmt.Println("volume", price1, price2, 0, "outside high")
		return 0, nil
	}

	p1 := num.Max(pLow, price1)
	p2 := num.Min(price2, pHigh)

	// pick curve if
	cu := pool.lower
	if p1.GTE(base) {
		cu = pool.upper
	}

	// let calculate the volume between these two

	v1 := cu.impliedPosition(p1, pHigh)
	v2 := cu.impliedPosition(p2, pHigh)

	// need to use position to tell us if its buy or sell volume
	side := types.SideBuy
	if v1.LessThan(num.DecimalZero()) {
		side = types.SideSell
	}

	volume := v1.Sub(v2).Abs().IntPart()

	retPrice := price2
	if side == types.SideSell {
		retPrice = price1
	}
	fmt.Println("volume", price1, price2, volume, "good", retPrice, "side", side)
	return uint64(volume), retPrice.Clone()

}
