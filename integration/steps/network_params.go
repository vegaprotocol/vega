package steps

import (
	"context"

	"github.com/cucumber/godog/gherkin"

	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/netparams"
)

func TheFollowingNetworkParametersAreSet(exec *execution.Engine, netParams *netparams.Store, table *gherkin.DataTable) error {
	ctx := context.Background()
	var watchParams []netparams.WatchParam

	for _, row := range TableWrapper(*table).Parse() {
		name := row.MustStr("name")

		switch name {
		case netparams.MarketAuctionMinimumDuration:
			watchParams = append(watchParams, netparams.WatchParam{
				Param:   netparams.MarketAuctionMinimumDuration,
				Watcher: exec.OnMarketAuctionMinimumDurationUpdate,
			})
		}
	}

	if err := netParams.Watch(watchParams...); err != nil {
		return err
	}

	for _, row := range TableWrapper(*table).Parse() {
		name := row.MustStr("name")

		switch name {
		case netparams.MarketAuctionMinimumDuration:
			d := row.MustDurationSec("value")
			if err := netParams.Update(ctx, netparams.MarketAuctionMinimumDuration, d.String()); err != nil {
				return err
			}
		}
	}

	netParams.DispatchChanges(ctx)

	return nil
}