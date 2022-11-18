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
	"strconv"

	"github.com/cucumber/godog"

	"code.vegaprotocol.io/vega/core/netparams"
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
		case netparams.MarkPriceUpdateMaximumFrequency:
			f := row.MustDurationStr("value")
			str := f.String()
			if err := netParams.Update(ctx, netparams.MarkPriceUpdateMaximumFrequency, str); err != nil {
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
