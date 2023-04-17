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
	"fmt"
	"time"

	"github.com/cucumber/godog"

	"code.vegaprotocol.io/vega/core/plugins"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"
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

	// check position status if needed
	ps, checkPS := row.positionState()
	party := row.party()
	for retries > 0 {
		pos, err = positionService.GetPositionsByParty(party)
		if err != nil {
			return errCannotGetPositionForParty(party, err)
		}

		if areSamePosition(pos, row) {
			if !checkPS {
				return nil
			}
			// check state if required
			states, _ := positionService.GetPositionStatesByParty(party)
			if len(states) == 1 && states[0] == ps {
				return nil
			}
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
	}, []string{
		"status",
	})
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

func (r pnlRow) positionState() (vega.PositionStatus, bool) {
	if !r.row.HasColumn("status") {
		// we do not have the status column sepcified
		return vega.PositionStatus_POSITION_STATUS_UNSPECIFIED, false
	}
	return r.row.MustPositionStatus("status"), true
}
