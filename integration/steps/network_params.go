package steps

import (
	"context"
	"strconv"

	"github.com/cucumber/godog"

	"code.vegaprotocol.io/vega/netparams"
)

func TheFollowingNetworkParametersAreSet(netParams *netparams.Store, table *godog.Table) error {
	ctx := context.Background()
	for _, row := range parseNetworkParametersTable(table) {
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
		case netparams.MarketLiquidityTargetStakeTriggeringRatio:
			f := row.MustF64("value")
			n := strconv.FormatFloat(f, 'f', -1, 64)
			if err := netParams.Update(ctx, netparams.MarketLiquidityTargetStakeTriggeringRatio, n); err != nil {
				return err
			}
		case netparams.MarketTargetStakeTimeWindow:
			f := row.MustDurationStr("value")
			str := f.String()
			if err := netParams.Update(ctx, netparams.MarketTargetStakeTimeWindow, str); err != nil {
				return err
			}
		default:
			value := row.MustStr("value")
			if err := netParams.Update(ctx, name, value); err != nil {
				return err
			}
		}
	}

	netParams.DispatchChanges(ctx)

	return nil
}

func parseNetworkParametersTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"name",
		"value",
	}, []string{})
}
