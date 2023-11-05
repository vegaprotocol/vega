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
	"encoding/hex"
	"errors"
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/cucumber/godog"
)

type LPUpdate struct {
	MarketID         string
	CommitmentAmount *num.Uint
	Fee              num.Decimal
	Reference        string
	LpType           string
	Err              string
}

func PartyCancelsTheirLiquidityProvision(exec Execution, marketID, party string) error {
	cancel := types.LiquidityProvisionCancellation{
		MarketID: marketID,
	}
	if err := exec.CancelLiquidityProvision(context.Background(), &cancel, party); err != nil {
		return errCancelLiquidityProvision(party, marketID, err)
	}
	return nil
}

func PartiesSubmitLiquidityProvision(exec Execution, table *godog.Table) error {
	lps := map[string]*LPUpdate{}
	parties := map[string]string{}
	keys := []string{}

	var errRow ErroneousRow
	for _, r := range parseSubmitLiquidityProvisionTable(table) {
		row := submitLiquidityProvisionRow{row: r}
		if errRow == nil || row.ExpectError() {
			errRow = row
		}
		id := row.ID()
		ref := id

		if _, ok := lps[id]; !ok {
			lp := &LPUpdate{
				MarketID:         row.MarketID(),
				CommitmentAmount: row.CommitmentAmount(),
				Fee:              row.Fee(),
				Reference:        ref,
				LpType:           row.LpType(),
				Err:              row.Error(),
			}
			parties[id] = row.Party()
			lps[id] = lp
			keys = append(keys, id)
		}
	}
	// ensure we always submit in the same order
	sort.Strings(keys)
	for _, id := range keys {
		lp, ok := lps[id]
		if !ok {
			return errors.New("LP  not found")
		}
		party, ok := parties[id]
		if !ok {
			return errors.New("party for LP not found")
		}

		if lp.LpType == "amendment" {
			lpa := &types.LiquidityProvisionAmendment{
				MarketID:         lp.MarketID,
				CommitmentAmount: lp.CommitmentAmount,
				Fee:              lp.Fee,
				Reference:        lp.Reference,
			}

			err := exec.AmendLiquidityProvision(context.Background(), lpa, party)
			if ceerr := checkExpectedError(errRow, err, errAmendingLiquidityProvision(lpa, party, err)); ceerr != nil {
				return ceerr
			}
		} else if lp.LpType == "submission" {
			sub := &types.LiquidityProvisionSubmission{
				MarketID:         lp.MarketID,
				CommitmentAmount: lp.CommitmentAmount,
				Fee:              lp.Fee,
				Reference:        lp.Reference,
			}
			deterministicID := hex.EncodeToString(crypto.Hash([]byte(id + party + lp.MarketID)))
			err := exec.SubmitLiquidityProvision(context.Background(), sub, party, id, deterministicID)
			if ceerr := checkExpectedError(errRow, err, errSubmittingLiquidityProvision(sub, party, id, err)); ceerr != nil {
				return ceerr
			}
		} else if lp.LpType == "cancellation" {
			cancel := types.LiquidityProvisionCancellation{
				MarketID: lp.MarketID,
			}
			err := exec.CancelLiquidityProvision(context.Background(), &cancel, party)
			if ceerr := checkExpectedError(errRow, err, errCancelLiquidityProvision(party, lp.MarketID, err)); ceerr != nil {
				return ceerr
			}
		}
	}
	return nil
}

func errSubmittingLiquidityProvision(lp *types.LiquidityProvisionSubmission, party, id string, err error) error {
	return fmt.Errorf("failed to submit [%v] for party %s and id %s: %v", lp, party, id, err)
}

func errCancelLiquidityProvision(party, market string, err error) error {
	return fmt.Errorf("failed to cancel LP for party %s on market %s: %v", party, market, err)
}

func errAmendingLiquidityProvision(lp *types.LiquidityProvisionAmendment, party string, err error) error {
	return fmt.Errorf("failed to amend [%v] for party %s : %v", lp, party, err)
}

func parseSubmitLiquidityProvisionTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"id",
		"party",
		"market id",
		"commitment amount",
		"fee",
		"lp type",
	}, []string{
		"reference",
		"error",
	})
}

type submitLiquidityProvisionRow struct {
	row RowWrapper
}

func (r submitLiquidityProvisionRow) ID() string {
	return r.row.MustStr("id")
}

func (r submitLiquidityProvisionRow) Party() string {
	return r.row.MustStr("party")
}

func (r submitLiquidityProvisionRow) MarketID() string {
	return r.row.MustStr("market id")
}

func (r submitLiquidityProvisionRow) Side() types.Side {
	if len(r.row.Str("side")) == 0 {
		return types.SideUnspecified
	}
	return r.row.MustSide("side")
}

func (r submitLiquidityProvisionRow) CommitmentAmount() *num.Uint {
	return r.row.MustUint("commitment amount")
}

func (r submitLiquidityProvisionRow) Fee() num.Decimal {
	return r.row.MustDecimal("fee")
}

func (r submitLiquidityProvisionRow) LpType() string {
	return r.row.MustStr("lp type")
}

func (r submitLiquidityProvisionRow) Reference() string {
	return r.row.Str("reference")
}

func (r submitLiquidityProvisionRow) Error() string {
	return r.row.Str("error")
}

func (r submitLiquidityProvisionRow) ExpectError() bool {
	return r.row.HasColumn("error")
}
