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

package steps

import (
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/core/plugins"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/cucumber/godog"
)

func PartiesHaveTheFollowingProfitAndLoss(
	exec Execution,
	positionService *plugins.Positions,
	table *godog.Table,
) error {
	for _, r := range parseProfitAndLossTable(table) {
		row := pnlRow{row: r}
		if err := positionAPIProduceTheFollowingRow(exec, positionService, row); err != nil {
			return err
		}
	}
	return nil
}

func positionAPIProduceTheFollowingRow(exec Execution, positionService *plugins.Positions, row pnlRow) (err error) {
	retries := 2
	sleepTimeMs := 100

	var pos []*types.Position
	// check position status if needed
	ps, checkPS := row.positionState()
	checkFees := row.checkFees()
	checkFunding := row.checkFunding()
	party := row.party()
	readableParty := party
	if row.isAMM() {
		id, ok := exec.GetAMMSubAccountID(party)
		if !ok {
			return errCannotGetPositionForParty(party, fmt.Errorf("vAMM alias %s not found", party))
		}
		party = id
	}
	for retries > 0 {
		if len(row.market()) > 0 {
			p, err := positionService.GetPositionsByMarketAndParty(row.market(), party)
			pos = []*types.Position{p}
			if err != nil {
				if row.volume() == 0 && row.realisedPNL().IsZero() && row.unrealisedPNL().IsZero() {
					return nil
				}
				return errCannotGetPositionForParty(readableParty, err)
			}
		} else {
			pos, err = positionService.GetPositionsByParty(party)
		}

		if err != nil {
			if row.volume() == 0 && row.realisedPNL().IsZero() && row.unrealisedPNL().IsZero() {
				return nil
			}
			return errCannotGetPositionForParty(readableParty, err)
		}

		if areSamePosition(pos, row) {
			if !checkFees && !checkFunding && !checkPS {
				return nil
			}
			match := true
			if checkFees {
				match = feesMatch(pos, row)
			}
			if checkFunding && match {
				match = fundingMatches(pos, row)
			}
			if checkPS && match {
				// check state if required
				states, _ := positionService.GetPositionStatesByParty(party)
				match = len(states) == 1 && states[0] == ps
			}
			if match {
				return nil
			}
		}
		time.Sleep(time.Duration(sleepTimeMs) * time.Millisecond)
		sleepTimeMs *= 2
		retries--
	}

	if len(pos) == 0 {
		if row.volume() == 0 && row.realisedPNL().IsZero() && row.unrealisedPNL().IsZero() {
			return nil
		}
		return errNoPositionForMarket(row.party())
	}

	return errProfitAndLossValuesForParty(pos, row)
}

func errProfitAndLossValuesForParty(pos []*types.Position, row pnlRow) error {
	if pos[0] == nil {
		pos[0] = &types.Position{}
	}
	return formatDiff(
		fmt.Sprintf("invalid positions values for party(%v)", row.party()),
		row.diffMap(),
		map[string]string{
			"volume":                 i64ToS(pos[0].OpenVolume),
			"unrealised PNL":         pos[0].UnrealisedPnl.String(),
			"realised PNL":           pos[0].RealisedPnl.String(),
			"taker fees":             pos[0].TakerFeesPaid.String(),
			"taker fees since":       pos[0].TakerFeesPaidSince.String(),
			"maker fees":             pos[0].MakerFeesReceived.String(),
			"maker fees since":       pos[0].MakerFeesReceivedSince.String(),
			"other fees":             pos[0].FeesPaid.String(),
			"other fees since":       pos[0].FeesPaidSince.String(),
			"funding payments":       pos[0].FundingPaymentAmount.String(),
			"funding payments since": pos[0].FundingPaymentAmountSince.String(),
		},
	)
}

func errNoPositionForMarket(party string) error {
	return fmt.Errorf("party do not have a position, party(%v)", party)
}

func areSamePosition(pos []*types.Position, row pnlRow) bool {
	return len(pos) == 1 &&
		pos[0].OpenVolume == row.volume() &&
		pos[0].RealisedPnl.Equals(row.realisedPNL()) &&
		pos[0].UnrealisedPnl.Equals(row.unrealisedPNL())
}

func feesMatch(pos []*types.Position, row pnlRow) bool {
	if len(pos) == 0 {
		return false
	}
	taker, ok := row.takerFees()
	if ok && !taker.EQ(pos[0].TakerFeesPaid) {
		return false
	}
	maker, ok := row.makerFees()
	if ok && !maker.EQ(pos[0].MakerFeesReceived) {
		return false
	}
	other, ok := row.otherFees()
	if ok && !other.EQ(pos[0].FeesPaid) {
		return false
	}
	taker, ok = row.takerFeesSince()
	if ok && !taker.EQ(pos[0].TakerFeesPaidSince) {
		return false
	}
	maker, ok = row.makerFeesSince()
	if ok && !maker.EQ(pos[0].MakerFeesReceivedSince) {
		return false
	}
	other, ok = row.otherFeesSince()
	if ok && !other.EQ(pos[0].FeesPaidSince) {
		return false
	}
	return true
}

func fundingMatches(pos []*types.Position, row pnlRow) bool {
	if len(pos) == 0 {
		return false
	}
	fp, ok := row.fundingPayment()
	if ok && !fp.EQ(pos[0].FundingPaymentAmount) {
		return false
	}
	fp, ok = row.fundingPaymentSince()
	if ok && !fp.EQ(pos[0].FundingPaymentAmountSince) {
		return false
	}
	return true
}

func errCannotGetPositionForParty(party string, err error) error {
	return fmt.Errorf("error getting party position, party(%v), err(%v)", party, err)
}

func parseProfitAndLossTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"volume",
		"unrealised pnl",
		"realised pnl",
	}, []string{
		"status",
		"market id",
		"is amm",
		"taker fees",
		"taker fees since",
		"maker fees",
		"maker fees since",
		"other fees",
		"other fees since",
		"funding payments",
		"funding payments since",
	})
}

type pnlRow struct {
	row RowWrapper
}

func (r pnlRow) party() string {
	return r.row.MustStr("party")
}

func (r pnlRow) market() string {
	if r.row.HasColumn("market id") {
		return r.row.MustStr("market id")
	}
	return ""
}

func (r pnlRow) volume() int64 {
	return r.row.MustI64("volume")
}

func (r pnlRow) unrealisedPNL() num.Decimal {
	return r.row.MustDecimal("unrealised pnl")
}

func (r pnlRow) realisedPNL() num.Decimal {
	return r.row.MustDecimal("realised pnl")
}

func (r pnlRow) positionState() (vega.PositionStatus, bool) {
	if !r.row.HasColumn("status") {
		// we do not have the status column sepcified
		return vega.PositionStatus_POSITION_STATUS_UNSPECIFIED, false
	}
	return r.row.MustPositionStatus("status"), true
}

func (r pnlRow) isAMM() bool {
	if !r.row.HasColumn("is amm") {
		return false
	}
	return r.row.MustBool("is amm")
}

func (r pnlRow) takerFees() (*num.Uint, bool) {
	if !r.row.HasColumn("taker fees") {
		return nil, false
	}
	return r.row.MustUint("taker fees"), true
}

func (r pnlRow) takerFeesSince() (*num.Uint, bool) {
	if !r.row.HasColumn("taker fees since") {
		return nil, false
	}
	return r.row.MustUint("taker fees since"), true
}

func (r pnlRow) makerFees() (*num.Uint, bool) {
	if !r.row.HasColumn("maker fees") {
		return nil, false
	}
	return r.row.MustUint("maker fees"), true
}

func (r pnlRow) makerFeesSince() (*num.Uint, bool) {
	if !r.row.HasColumn("maker fees since") {
		return nil, false
	}
	return r.row.MustUint("maker fees since"), true
}

func (r pnlRow) otherFees() (*num.Uint, bool) {
	if !r.row.HasColumn("other fees") {
		return nil, false
	}
	return r.row.MustUint("other fees"), true
}

func (r pnlRow) otherFeesSince() (*num.Uint, bool) {
	if !r.row.HasColumn("other fees since") {
		return nil, false
	}
	return r.row.MustUint("other fees since"), true
}

func (r pnlRow) fundingPayment() (*num.Int, bool) {
	if !r.row.HasColumn("funding payments") {
		return nil, false
	}
	return r.row.MustInt("funding payments"), true
}

func (r pnlRow) fundingPaymentSince() (*num.Int, bool) {
	if !r.row.HasColumn("funding payments since") {
		return nil, false
	}
	return r.row.MustInt("funding payments since"), true
}

func (r pnlRow) checkFees() bool {
	if _, taker := r.takerFees(); taker {
		return true
	}
	if _, maker := r.makerFees(); maker {
		return true
	}
	if _, other := r.otherFees(); other {
		return true
	}
	if _, ok := r.takerFeesSince(); ok {
		return true
	}
	if _, ok := r.makerFeesSince(); ok {
		return true
	}
	if _, ok := r.otherFeesSince(); ok {
		return true
	}
	return false
}

func (r pnlRow) checkFunding() bool {
	if _, ok := r.fundingPayment(); ok {
		return true
	}
	_, ok := r.fundingPaymentSince()
	return ok
}

func (r pnlRow) diffMap() map[string]string {
	m := map[string]string{
		"volume":                 i64ToS(r.volume()),
		"unrealised PNL":         r.unrealisedPNL().String(),
		"realised PNL":           r.realisedPNL().String(),
		"taker fees":             "",
		"taker fees since":       "",
		"maker fees":             "",
		"maker fees since":       "",
		"other fees":             "",
		"other fees since":       "",
		"funding payments":       "",
		"funding payments since": "",
	}
	if v, ok := r.takerFees(); ok {
		m["taker fees"] = v.String()
	}
	if v, ok := r.makerFees(); ok {
		m["maker fees"] = v.String()
	}
	if v, ok := r.otherFees(); ok {
		m["other fees"] = v.String()
	}
	if v, ok := r.takerFeesSince(); ok {
		m["taker fees since"] = v.String()
	}
	if v, ok := r.makerFeesSince(); ok {
		m["maker fees since"] = v.String()
	}
	if v, ok := r.otherFeesSince(); ok {
		m["other fees since"] = v.String()
	}
	if v, ok := r.fundingPayment(); ok {
		m["funding payments"] = v.String()
	}
	if v, ok := r.fundingPaymentSince(); ok {
		m["funding payments since"] = v.String()
	}
	return m
}
