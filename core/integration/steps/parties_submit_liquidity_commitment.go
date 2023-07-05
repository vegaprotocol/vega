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

package steps

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/libs/crypto"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"github.com/cucumber/godog"
)

type LP2Update struct {
	MarketID         string
	CommitmentAmount *num.Uint
	Fee              num.Decimal
	Reference        string
	LpType           string
	Err              string
}

func PartiesSubmitLiquidityCommitment(exec Execution, table *godog.Table) error {
	lps := map[string]*LP2Update{}
	parties := map[string]string{}
	keys := []string{}

	// var clp *types.LiquidityProvisionSubmission
	// checkAmt := num.NewUint(50000000)
	var errRow ErroneousRow
	for _, r := range parseSubmitLiquidityCommitmentTable(table) {
		row := submitLiquidityProvisionRow{row: r}
		if errRow == nil || row.ExpectError() {
			errRow = row
		}
		id := row.ID()
		ref := id

		lp, ok := lps[id]
		if !ok {
			lp = &LP2Update{
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
			// TODO: Check the type when SLA is merged
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
			// TODO: Check the type when SLA is merged
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
			// TODO: Check the type when SLA is merged
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

func errSubmittingLiquidityCommitment(lp *types.LiquidityProvisionSubmission, party, id string, err error) error {
	return fmt.Errorf("failed to submit [%v] for party %s and id %s: %v", lp, party, id, err)
}

func parseSubmitLiquidityCommitmentTable(table *godog.Table) []RowWrapper {
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

type submitLiquidityCommitmentRow struct {
	row RowWrapper
}

func (r submitLiquidityCommitmentRow) ID() string {
	return r.row.MustStr("id")
}

func (r submitLiquidityCommitmentRow) Party() string {
	return r.row.MustStr("party")
}

func (r submitLiquidityCommitmentRow) MarketID() string {
	return r.row.MustStr("market id")
}

func (r submitLiquidityCommitmentRow) CommitmentAmount() *num.Uint {
	return r.row.MustUint("commitment amount")
}

func (r submitLiquidityCommitmentRow) Fee() num.Decimal {
	return r.row.MustDecimal("fee")
}

func (r submitLiquidityCommitmentRow) LpType() string {
	return r.row.MustStr("lp type")
}

func (r submitLiquidityCommitmentRow) Reference() string {
	return r.row.Str("reference")
}

func (r submitLiquidityCommitmentRow) Error() string {
	return r.row.Str("error")
}

func (r submitLiquidityCommitmentRow) ExpectError() bool {
	return r.row.HasColumn("error")
}
