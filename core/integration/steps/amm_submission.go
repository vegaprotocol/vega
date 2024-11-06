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
			r:       r,
			isAmend: true,
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
		"party",        // str
		"market id",    // str
		"amount",       // uint
		"slippage",     // dec
		"base",         // uint
		"proposed fee", // dec
	}, []string{
		"lower bound",    // uint
		"upper bound",    // uint
		"lower leverage", // dec
		"upper leverage", // dec
		"data source id",
		"minimum price change trigger",
		"spread",
		"error",
	})
}

func parseAmendAMMTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",     // str
		"market id", // str
		"slippage",  // dec
	}, []string{
		"proposed fee",   // dec
		"amount",         // uint
		"base",           // uint
		"lower bound",    // uint
		"upper bound",    // uint
		"lower leverage", // dec
		"upper leverage", // dec
		"data source id",
		"minimum price change trigger",
		"spread",
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
	r       RowWrapper
	isAmend bool
}

func (a ammRow) toSubmission() *types.SubmitAMM {
	if !a.r.HasColumn("lower bound") && !a.r.HasColumn("upper bound") {
		panic("required at least one upper bound and lower bound")
	}

	return &types.SubmitAMM{
		AMMBaseCommand: types.AMMBaseCommand{
			MarketID:                  a.marketID(),
			Party:                     a.party(),
			SlippageTolerance:         a.slippage(),
			ProposedFee:               a.proposedFee(),
			MinimumPriceChangeTrigger: a.minimumPriceChangeTrigger(),
			Spread:                    a.spread(),
		},
		CommitmentAmount: a.amount(),
		Parameters: &types.ConcentratedLiquidityParameters{
			Base:                 a.base(),
			LowerBound:           a.lowerBound(),
			UpperBound:           a.upperBound(),
			LeverageAtLowerBound: a.lowerLeverage(),
			LeverageAtUpperBound: a.upperLeverage(),
			DataSourceID:         a.dataSourceID(),
		},
	}
}

func (a ammRow) toAmendment() *types.AmendAMM {
	ret := &types.AmendAMM{
		AMMBaseCommand: types.AMMBaseCommand{
			MarketID:                  a.marketID(),
			Party:                     a.party(),
			SlippageTolerance:         a.slippage(),
			ProposedFee:               a.proposedFee(),
			MinimumPriceChangeTrigger: a.minimumPriceChangeTrigger(),
			Spread:                    a.spread(),
		},
	}
	if a.r.HasColumn("amount") {
		ret.CommitmentAmount = a.amount()
	}
	params := &types.ConcentratedLiquidityParameters{
		DataSourceID: a.dataSourceID(),
	}
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
	if a.r.HasColumn("lower leverage") {
		params.LeverageAtLowerBound = a.lowerLeverage()
		paramSet = true
	}
	if a.r.HasColumn("upper leverage") {
		params.LeverageAtUpperBound = a.upperLeverage()
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

func (a ammRow) proposedFee() num.Decimal {
	if !a.isAmend {
		return a.r.MustDecimal("proposed fee")
	}

	if a.r.HasColumn("proposed fee") {
		return a.r.MustDecimal("proposed fee")
	}
	return num.DecimalZero()
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
	if !a.r.HasColumn("lower bound") {
		return nil
	}
	return a.r.MustUint("lower bound")
}

func (a ammRow) upperBound() *num.Uint {
	if !a.r.HasColumn("upper bound") {
		return nil
	}
	return a.r.MustUint("upper bound")
}

func (a ammRow) dataSourceID() *string {
	if !a.r.HasColumn("data source id") {
		return nil
	}
	return ptr.From(a.r.Str("data source id"))
}

func (a ammRow) minimumPriceChangeTrigger() num.Decimal {
	if !a.r.HasColumn("minimum price change trigger") {
		return num.DecimalZero()
	}
	return a.r.MustDecimal("minimum price change trigger")
}

func (a ammRow) spread() num.Decimal {
	if !a.r.HasColumn("spread") {
		return num.DecimalZero()
	}
	return a.r.MustDecimal("spread")
}

func (a ammRow) lowerLeverage() *num.Decimal {
	if !a.r.HasColumn("lower leverage") {
		return nil
	}
	return ptr.From(a.r.MustDecimal("lower leverage"))
}

func (a ammRow) upperLeverage() *num.Decimal {
	if !a.r.HasColumn("upper leverage") {
		return nil
	}
	return ptr.From(a.r.MustDecimal("upper leverage"))
}

func (a ammRow) method() types.AMMCancellationMethod {
	if !a.r.HasColumn("method") {
		return types.AMMCancellationMethodUnspecified
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
