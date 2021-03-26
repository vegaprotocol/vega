package steps

import (
	"context"
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/execution"
	types "code.vegaprotocol.io/vega/proto"
	"github.com/cucumber/godog/gherkin"
)

func TradersSubmitLiquidityProvision(exec *execution.Engine, table *gherkin.DataTable) error {
	lps := map[string]*types.LiquidityProvisionSubmission{}
	parties := map[string]string{}

	for _, row := range TableWrapper(*table).Parse() {
		id := row.Str("id")
		party := row.Str("party")
		marketID := row.Str("market id")
		amount := row.U64("commitment amount")
		fee := row.Str("fee")
		side := row.Side("order side")
		reference := row.PeggedReference("order reference")
		proportion := row.U32("order proportion")
		offset := row.I64("order offset")
		orderRefernce := row.Str("reference")

		if id == "id" {
			continue
		}

		lp, ok := lps[id]
		if !ok {
			lp = &types.LiquidityProvisionSubmission{
				MarketId:         marketID,
				CommitmentAmount: amount,
				Fee:              fee,
				Sells:            []*types.LiquidityOrder{},
				Buys:             []*types.LiquidityOrder{},
				Reference:        orderRefernce,
			}
			parties[id] = party
			lps[id] = lp
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
	for id, sub := range lps {
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

func errSubmittingLiquidityProvision(lp *types.LiquidityProvisionSubmission, party, id string, err error) error {
	return fmt.Errorf("failed to submit [%v] for party %s and id %s: %v", lp, party, id, err)
}
