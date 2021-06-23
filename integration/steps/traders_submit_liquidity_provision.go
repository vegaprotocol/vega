package steps

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/execution"
	types "code.vegaprotocol.io/vega/proto"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	"github.com/cucumber/godog/gherkin"
)

func TradersSubmitLiquidityProvision(exec *execution.Engine, table *gherkin.DataTable) error {
	lps := map[string]*commandspb.LiquidityProvisionSubmission{}
	parties := map[string]string{}
	keys := []string{}

	for _, r := range parseSubmitLiquidityProvisionTable(table) {
		row := submitLiquidityProvisionRow{row: r}

		id := row.ID()

		lp, ok := lps[id]
		if !ok {
			lp = &commandspb.LiquidityProvisionSubmission{
				MarketId:         row.MarketID(),
				CommitmentAmount: row.CommitmentAmount(),
				Fee:              row.Fee(),
				Sells:            []*types.LiquidityOrder{},
				Buys:             []*types.LiquidityOrder{},
				Reference:        row.Reference(),
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
		if row.Side() == types.Side_SIDE_BUY {
			lp.Buys = append(lp.Buys, lo)
		} else {
			lp.Sells = append(lp.Sells, lo)
		}
	}

	// ensure we always submit in the same order
	sort.Strings(keys)
	for _, id := range keys {
		sub := lps[id]
		party, ok := parties[id]
		if !ok {
			return errors.New("party for LP not found")
		}
		if err := exec.SubmitLiquidityProvision(context.Background(), sub, party, id); err != nil {
			return errSubmittingLiquidityProvision(sub, party, id, err)
		}
	}
	return nil
}

func errSubmittingLiquidityProvision(lp *commandspb.LiquidityProvisionSubmission, party, id string, err error) error {
	return fmt.Errorf("failed to submit [%v] for party %s and id %s: %v", lp, party, id, err)
}

func parseSubmitLiquidityProvisionTable(table *gherkin.DataTable) []RowWrapper {
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

func (r submitLiquidityProvisionRow) CommitmentAmount() uint64 {
	return r.row.MustU64("commitment amount")
}

func (r submitLiquidityProvisionRow) Fee() string {
	return r.row.MustStr("fee")
}

func (r submitLiquidityProvisionRow) Offset() int64 {
	return r.row.MustI64("offset")
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
