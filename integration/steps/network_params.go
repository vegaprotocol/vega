package steps

import (
	"context"
	"fmt"
	"strconv"

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
		case netparams.MarketTargetStakeTimeWindow:
			watchParams = append(watchParams, netparams.WatchParam{
				Param:   netparams.MarketTargetStakeTimeWindow,
				Watcher: exec.OnMarketTargetStakeTimeWindowUpdate,
			})
		case netparams.MarketTargetStakeScalingFactor:
			watchParams = append(watchParams, netparams.WatchParam{
				Param:   netparams.MarketTargetStakeScalingFactor,
				Watcher: exec.OnMarketTargetStakeScalingFactorUpdate,
			})
		default:
			return fmt.Errorf("unimplemented network param %v in feature test", name)
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
		case netparams.MarketTargetStakeTimeWindow:
			d := row.MustDurationSec("value")
			if err := netParams.Update(ctx, netparams.MarketTargetStakeTimeWindow, d.String()); err != nil {
				return err
			}
		case netparams.MarketTargetStakeScalingFactor:
			f := row.MustF64("value")
			n := strconv.FormatFloat(f, 'f', -1, 64)
			if err := netParams.Update(ctx, netparams.MarketTargetStakeScalingFactor, n); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unimplemented network param %v in feature test", name)
		}
	}

	netParams.DispatchChanges(ctx)

	return nil
}
