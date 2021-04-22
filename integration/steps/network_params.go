package steps

import (
	"context"
	"strconv"

	"github.com/cucumber/godog/gherkin"

	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/netparams"
)

func TheFollowingNetworkParametersAreSet(exec *execution.Engine, netParams *netparams.Store, table *gherkin.DataTable) error {
	ctx := context.Background()
	for _, row := range TableWrapper(*table).Parse() {
		name := row.MustStr("name")

		switch name {
		case netparams.MarketAuctionMinimumDuration:
			d := row.MustDurationSec("value")
			if err := netParams.Update(ctx, netparams.MarketAuctionMinimumDuration, d.String()); err != nil {
				return err
			}
		case netparams.MarketTargetStakeScalingFactor:
			f := row.MustF64("value")
			n := strconv.FormatFloat(f, 'f', -1, 64)
			if err := netParams.Update(ctx, netparams.MarketTargetStakeScalingFactor, n); err != nil {
				return err
			}
		default:
			value := row.MustStr("value")
			if err := netParams.Update(ctx, name, value); err != nil {
				return err
			}
			// return fmt.Errorf("unimplemented network param %v in feature test", name)
		}
	}

	netParams.DispatchChanges(ctx)

	return nil
}
