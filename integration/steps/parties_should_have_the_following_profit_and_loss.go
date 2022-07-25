// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package steps

import (
	"fmt"
	"time"

	"github.com/cucumber/godog"

	"code.vegaprotocol.io/vega/plugins"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

func PartiesHaveTheFollowingProfitAndLoss(
	positionService *plugins.Positions,
	table *godog.Table,
) error {
	for _, r := range parseProfitAndLossTable(table) {
		row := pnlRow{row: r}
		if err := positionAPIProduceTheFollowingRow(positionService, row); err != nil {
			return err
		}
	}
	return nil
}

func positionAPIProduceTheFollowingRow(positionService *plugins.Positions, row pnlRow) (err error) {
	retries := 2
	sleepTimeMs := 100

	var pos []*types.Position
	for retries > 0 {
		pos, err = positionService.GetPositionsByParty(row.party())
		if err != nil {
			return errCannotGetPositionForParty(row.party(), err)
		}

		if areSamePosition(pos, row) {
			return nil
		}

		time.Sleep(time.Duration(sleepTimeMs) * time.Millisecond)
		sleepTimeMs *= 2
		retries--
	}

	if len(pos) == 0 {
		return errNoPositionForMarket(row.party())
	}

	return errProfitAndLossValuesForParty(pos, row)
}

func errProfitAndLossValuesForParty(pos []*types.Position, row pnlRow) error {
	return formatDiff(
		fmt.Sprintf("invalid positions values for party(%v)", row.party()),
		map[string]string{
			"volume":         i64ToS(row.volume()),
			"unrealised PNL": row.unrealisedPNL().String(),
			"realised PNL":   row.realisedPNL().String(),
		},
		map[string]string{
			"volume":         i64ToS(pos[0].OpenVolume),
			"unrealised PNL": pos[0].UnrealisedPnl.String(),
			"realised PNL":   pos[0].RealisedPnl.String(),
		},
	)
}

func errNoPositionForMarket(party string) error {
	return fmt.Errorf("party do not have a position, party(%v)", party)
}

func areSamePosition(pos []*types.Position, row pnlRow) bool {
	return len(pos) == 1 &&
		pos[0].OpenVolume == row.volume() &&
		pos[0].RealisedPnl.Equals(row.realisedPNL()) &&
		pos[0].UnrealisedPnl.Equals(row.unrealisedPNL())
}

func errCannotGetPositionForParty(party string, err error) error {
	return fmt.Errorf("error getting party position, party(%v), err(%v)", party, err)
}

func parseProfitAndLossTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"volume",
		"unrealised pnl",
		"realised pnl",
	}, []string{})
}

type pnlRow struct {
	row RowWrapper
}

func (r pnlRow) party() string {
	return r.row.MustStr("party")
}

func (r pnlRow) volume() int64 {
	return r.row.MustI64("volume")
}

func (r pnlRow) unrealisedPNL() num.Decimal {
	return r.row.MustDecimal("unrealised pnl")
}

func (r pnlRow) realisedPNL() num.Decimal {
	return r.row.MustDecimal("realised pnl")
}
