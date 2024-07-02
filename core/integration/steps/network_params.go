// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package steps

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/logging"

	"github.com/cucumber/godog"
)

var unwatched = map[string]struct{}{}

func DebugNetworkParameter(log *logging.Logger, netParams *netparams.Store, key string) error {
	value, err := netParams.Get(key)
	if err != nil {
		return err
	}
	log.Info(fmt.Sprintf("\n\n%s: %s\n", key, value))
	return nil
}

func TheFollowingNetworkParametersAreSet(netParams *netparams.Store, table *godog.Table) error {
	ctx := context.Background()
	for _, row := range parseNetworkParametersTable(table) {
		name := row.MustStr("name")

		if _, ok := unwatched[name]; !ok && !netParams.AnyWatchers(name) {
			return errNoWatchersSpecified(name)
		}

		switch name {
		case netparams.MarketAuctionMinimumDuration:
			d := row.MustDurationSec("value")
			if err := netParams.Update(ctx, netparams.MarketAuctionMinimumDuration, d.String()); err != nil {
				return err
			}
		case netparams.MarketAuctionMaximumDuration:
			d := row.MustDurationStr("value")
			if err := netParams.Update(ctx, netparams.MarketAuctionMaximumDuration, d.String()); err != nil {
				return err
			}
		case netparams.MarkPriceUpdateMaximumFrequency:
			f := row.MustDurationStr("value")
			str := f.String()
			if err := netParams.Update(ctx, netparams.MarkPriceUpdateMaximumFrequency, str); err != nil {
				return err
			}
		case netparams.InternalCompositePriceUpdateFrequency:
			f := row.MustDurationStr("value")
			str := f.String()
			if err := netParams.Update(ctx, netparams.InternalCompositePriceUpdateFrequency, str); err != nil {
				return err
			}
		case netparams.MarketLiquidityEquityLikeShareFeeFraction:
			dv := row.MustDecimal("value")
			if err := netParams.Update(ctx, netparams.MarketLiquidityEquityLikeShareFeeFraction, dv.String()); err != nil {
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
