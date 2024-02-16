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
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"

	"github.com/cucumber/godog"
)

func PartiesSubmitTheFollowingAMMs(exec Execution, table *godog.Table) error {
	ctx := context.Background()
	for _, r := range parseSubmitAMMTable(table) {
		row := ammRow{
			r: r,
		}
		fail, eStr := row.err()
		if err := exec.SubmitAMM(ctx, row.toSubmission()); err != nil {
			if !fail {
				return err
			}
			if err.Error() != eStr {
				return fmt.Errorf("expected error %s, instead got: %s (%v)", eStr, err.Error(), err)
			}
		}
	}
	return nil
}

func PartiesAmendTheFollowingAMMs(exec Execution, table *godog.Table) error {
	ctx := context.Background()
	for _, r := range parseAmendAMMTable(table) {
		row := ammRow{
			r: r,
		}
		fail, eStr := row.err()
		if err := exec.AmendAMM(ctx, row.toAmendment()); err != nil {
			if !fail {
				return err
			}
			if err.Error() != eStr {
				return fmt.Errorf("expected error %s, instead got: %s (%v)", eStr, err.Error(), err)
			}
		}
	}
	return nil
}

func PartiesCancelTheFollowingAMMs(exec Execution, table *godog.Table) error {
	ctx := context.Background()
	for _, r := range parseCancelAMMTable(table) {
		row := ammRow{
			r: r,
		}
		fail, eStr := row.err()
		if err := exec.CancelAMM(ctx, row.toCancel()); err != nil {
			if !fail {
				return err
			}
			if err.Error() != eStr {
				return fmt.Errorf("expected error %s, instead got: %s (%v)", eStr, err.Error(), err)
			}
		}
	}
	return nil
}

func parseSubmitAMMTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",              // str
		"market id",          // str
		"amount",             // uint
		"slippage",           // dec
		"base",               // uint
		"lower bound",        // uint
		"upper bound",        // uint
		"lower margin ratio", // dec
		"upper margin ratio", // dec
	}, []string{
		"error",
	})
}

func parseAmendAMMTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",     // str
		"market id", // str
		"slippage",  // dec
	}, []string{
		"amount",             // uint
		"base",               // uint
		"lower bound",        // uint
		"upper bound",        // uint
		"lower margin ratio", // dec
		"upper margin ratio", // dec
		"error",
	})
}

func parseCancelAMMTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"market id",
		"method",
	}, []string{
		"error",
	})
}

type ammRow struct {
	r RowWrapper
}

func (a ammRow) toSubmission() *types.SubmitAMM {
	return &types.SubmitAMM{
		AMMBaseCommand: types.AMMBaseCommand{
			MarketID:          a.marketID(),
			Party:             a.party(),
			SlippageTolerance: a.slippage(),
		},
		CommitmentAmount: a.amount(),
		Parameters: &types.ConcentratedLiquidityParameters{
			Base:                    a.base(),
			LowerBound:              a.lowerBound(),
			UpperBound:              a.upperBound(),
			MarginRatioAtLowerBound: ptr.From(a.lowerMargin()),
			MarginRatioAtUpperBound: ptr.From(a.upperMargin()),
		},
	}
}

func (a ammRow) toAmendment() *types.AmendAMM {
	ret := &types.AmendAMM{
		AMMBaseCommand: types.AMMBaseCommand{
			MarketID:          a.marketID(),
			Party:             a.party(),
			SlippageTolerance: a.slippage(),
		},
	}
	if a.r.HasColumn("amount") {
		ret.CommitmentAmount = a.amount()
	}
	params := &types.ConcentratedLiquidityParameters{}
	paramSet := false
	if a.r.HasColumn("base") {
		params.Base = a.base()
		paramSet = true
	}
	if a.r.HasColumn("lower bound") {
		params.LowerBound = a.lowerBound()
		paramSet = true
	}
	if a.r.HasColumn("upper bound") {
		params.UpperBound = a.upperBound()
		paramSet = true
	}
	if a.r.HasColumn("lower margin ratio") {
		params.MarginRatioAtLowerBound = ptr.From(a.lowerMargin())
		paramSet = true
	}
	if a.r.HasColumn("upper margin ratio") {
		params.MarginRatioAtUpperBound = ptr.From(a.upperMargin())
		paramSet = true
	}
	if paramSet {
		ret.Parameters = params
	}
	return ret
}

func (a ammRow) toCancel() *types.CancelAMM {
	return &types.CancelAMM{
		MarketID: a.marketID(),
		Party:    a.party(),
		Method:   a.method(),
	}
}

func (a ammRow) party() string {
	return a.r.MustStr("party")
}

func (a ammRow) marketID() string {
	return a.r.MustStr("market id")
}

func (a ammRow) amount() *num.Uint {
	return a.r.MustUint("amount")
}

func (a ammRow) slippage() num.Decimal {
	return a.r.MustDecimal("slippage")
}

func (a ammRow) base() *num.Uint {
	return a.r.MustUint("base")
}

func (a ammRow) lowerBound() *num.Uint {
	return a.r.MustUint("lower bound")
}

func (a ammRow) upperBound() *num.Uint {
	return a.r.MustUint("upper bound")
}

func (a ammRow) lowerMargin() num.Decimal {
	return a.r.MustDecimal("lower margin ratio")
}

func (a ammRow) upperMargin() num.Decimal {
	return a.r.MustDecimal("upper margin ratio")
}

func (a ammRow) method() types.AMMPoolCancellationMethod {
	if !a.r.HasColumn("method") {
		return types.AMMPoolCancellationMethodUnspecified
	}
	return a.r.MustAMMCancelationMethod("method")
}

func (a ammRow) err() (bool, string) {
	if !a.r.HasColumn("error") {
		return false, ""
	}
	str := a.r.MustStr("error")
	return true, str
}
