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
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/core/plugins"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/cucumber/godog"
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
		if len(row.market()) > 0 {
			p, err := positionService.GetPositionsByMarketAndParty(row.market(), party)
			pos = []*types.Position{p}
			if err != nil {
				return errCannotGetPositionForParty(party, err)
			}
		} else {
			pos, err = positionService.GetPositionsByParty(party)
		}

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
		"market id",
	})
}

type pnlRow struct {
	row RowWrapper
}

func (r pnlRow) party() string {
	return r.row.MustStr("party")
}

func (r pnlRow) market() string {
	if r.row.HasColumn("market id") {
		return r.row.MustStr("market id")
	}
	return ""
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
