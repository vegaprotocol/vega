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

	for _, row := range TableWrapper(*table).Parse() {
		id := row.MustStr("id")
		party := row.MustStr("party")
		marketID := row.MustStr("market id")
		amount := row.MustU64("commitment amount")
		fee := row.MustStr("fee")
		side := row.MustSide("order side")
		reference := row.MustPeggedReference("order reference")
		proportion := row.MustU32("order proportion")
		offset := row.MustI64("order offset")
		orderReference := row.Str("reference")

		if id == "id" {
			continue
		}

		lp, ok := lps[id]
		if !ok {
			lp = &commandspb.LiquidityProvisionSubmission{
				MarketId:         marketID,
				CommitmentAmount: amount,
				Fee:              fee,
				Sells:            []*types.LiquidityOrder{},
				Buys:             []*types.LiquidityOrder{},
				Reference:        orderReference,
			}
			parties[id] = party
			lps[id] = lp
			keys = append(keys, id)
		}
		lo := &types.LiquidityOrder{
			Reference:  reference,
			Proportion: proportion,
			Offset:     offset,
		}
		if side == types.Side_SIDE_BUY {
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
