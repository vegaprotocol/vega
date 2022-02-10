package steps

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/libs/crypto"

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/cucumber/godog"
)

type LPUpdate struct {
	MarketID         string
	CommitmentAmount *num.Uint
	Fee              num.Decimal
	Sells            []*types.LiquidityOrder
	Buys             []*types.LiquidityOrder
	Reference        string
	LpType           string
}

func PartiesSubmitLiquidityProvision(exec Execution, table *godog.Table) error {
	lps := map[string]*LPUpdate{}
	parties := map[string]string{}
	keys := []string{}

	// var clp *types.LiquidityProvisionSubmission
	// checkAmt := num.NewUint(50000000)
	for _, r := range parseSubmitLiquidityProvisionTable(table) {
		row := submitLiquidityProvisionRow{row: r}

		id := row.ID()
		ref := id

		lp, ok := lps[id]
		if !ok {
			lp = &LPUpdate{
				MarketID:         row.MarketID(),
				CommitmentAmount: row.CommitmentAmount(),
				Fee:              row.Fee(),
				Sells:            []*types.LiquidityOrder{},
				Buys:             []*types.LiquidityOrder{},
				Reference:        ref,
				LpType:           row.LpType(),
			}
			parties[id] = row.Party()
			lps[id] = lp
			keys = append(keys, id)
		}
		lo := &types.LiquidityOrder{
			Reference:  row.PeggedReference(),
			Proportion: row.Proportion(),
			Offset:     row.Offset(),
		}
		if row.Side() == types.SideBuy {
			lp.Buys = append(lp.Buys, lo)
		} else {
			lp.Sells = append(lp.Sells, lo)
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
				Sells:            lp.Sells,
				Buys:             lp.Buys,
				Reference:        lp.Reference,
			}
			if err := exec.AmendLiquidityProvision(context.Background(), lpa, party); err != nil {
				return errAmendingLiquidityProvision(lpa, party, err)
			}
		} else if lp.LpType == "submission" {
			sub := &types.LiquidityProvisionSubmission{
				MarketID:         lp.MarketID,
				CommitmentAmount: lp.CommitmentAmount,
				Fee:              lp.Fee,
				Sells:            lp.Sells,
				Buys:             lp.Buys,
				Reference:        lp.Reference,
			}

			if err := exec.SubmitLiquidityProvision(context.Background(), sub, party, id, crypto.RandomHash()); err != nil {
				return errSubmittingLiquidityProvision(sub, party, id, err)
			}
		}
	}
	return nil
}

func errSubmittingLiquidityProvision(lp *types.LiquidityProvisionSubmission, party, id string, err error) error {
	return fmt.Errorf("failed to submit [%v] for party %s and id %s: %v", lp, party, id, err)
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
		"side",
		"pegged reference",
		"proportion",
		"offset",
		"lp type",
	}, []string{
		"reference",
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
	return r.row.MustSide("side")
}

func (r submitLiquidityProvisionRow) CommitmentAmount() *num.Uint {
	return r.row.MustUint("commitment amount")
}

func (r submitLiquidityProvisionRow) Fee() num.Decimal {
	return r.row.MustDecimal("fee")
}

func (r submitLiquidityProvisionRow) Offset() *num.Uint {
	return r.row.MustUint("offset")
}

func (r submitLiquidityProvisionRow) LpType() string {
	return r.row.MustStr("lp type")
}

func (r submitLiquidityProvisionRow) Proportion() uint32 {
	return r.row.MustU32("proportion")
}

func (r submitLiquidityProvisionRow) PeggedReference() types.PeggedReference {
	return r.row.MustPeggedReference("pegged reference")
}

func (r submitLiquidityProvisionRow) Reference() string {
	return r.row.Str("reference")
}
