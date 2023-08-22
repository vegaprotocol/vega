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

package liquidity

import (
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"
)

func (e *Engine) GetAverageLiquidityScores() map[string]num.Decimal {
	return e.avgScores
}

func (e *Engine) UpdateAverageLiquidityScores(bestBid, bestAsk num.Decimal, minLpPrice, maxLpPrice *num.Uint) {
	current, total := e.GetCurrentLiquidityScores(bestBid, bestAsk, minLpPrice, maxLpPrice)
	nLps := len(current)
	if nLps == 0 {
		return
	}

	// normalise first
	equalFraction := num.DecimalOne().Div(num.DecimalFromInt64(int64(nLps)))
	for k, v := range current {
		if total.IsZero() {
			current[k] = equalFraction
		} else {
			current[k] = v.Div(total)
		}
	}

	if e.nAvg > 1 {
		n := num.DecimalFromInt64(e.nAvg)
		nMinusOneOverN := n.Sub(num.DecimalOne()).Div(n)

		for k, vNew := range current {
			// if not found then it defaults to 0
			vOld := e.avgScores[k]
			current[k] = vOld.Mul(nMinusOneOverN).Add(vNew.Div(n))
		}
	}

	for k := range current {
		current[k] = current[k].Round(10)
	}

	// always overwrite with latest to automatically remove LPs that are no longer ACTIVE from the list
	e.avgScores = current
	e.nAvg++
}

// GetCurrentLiquidityScores returns volume-weighted probability of trading per each LP's deployed orders.
func (e *Engine) GetCurrentLiquidityScores(bestBid, bestAsk num.Decimal, minLpPrice, maxLpPrice *num.Uint) (map[string]num.Decimal, num.Decimal) {
	provs := e.provisions.Slice()
	t := num.DecimalZero()
	r := make(map[string]num.Decimal, len(provs))
	for _, p := range provs {
		if p.Status != vega.LiquidityProvision_STATUS_ACTIVE {
			continue
		}
		orders := e.getAllActiveOrders(p.Party)
		l := e.suppliedEngine.CalculateLiquidityScore(orders, bestBid, bestAsk, minLpPrice, maxLpPrice)
		r[p.Party] = l
		t = t.Add(l)
	}
	return r, t
}

func (e *Engine) getAllActiveOrders(party string) []*types.Order {
	partyOrders := e.orderBook.GetOrdersPerParty(party)
	orders := make([]*types.Order, 0, len(partyOrders))
	for _, order := range partyOrders {
		if order.Status == vega.Order_STATUS_ACTIVE {
			orders = append(orders, order)
		}
	}
	return orders
}

func (e *Engine) ResetAverageLiquidityScores() {
	e.avgScores = make(map[string]num.Decimal, len(e.avgScores))
	e.nAvg = 1
}
