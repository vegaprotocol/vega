package steps

import (
	"fmt"
	"strconv"
	"time"

	"github.com/cucumber/godog/gherkin"

	"code.vegaprotocol.io/vega/plugins"
	types "code.vegaprotocol.io/vega/proto"
)

func TradersHaveTheFollowingProfitAndLoss(
	positionService *plugins.Positions,
	table *gherkin.DataTable,
) error {
	for _, r := range TableWrapper(*table).Parse() {
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
		pos, err = positionService.GetPositionsByParty(row.trader())
		if err != nil {
			return errCannotGetPositionForParty(row.trader(), err)
		}

		if areSamePosition(pos, row) {
			return nil
		}

		time.Sleep(time.Duration(sleepTimeMs) * time.Millisecond)
		sleepTimeMs *= 2
		retries--
	}

	if len(pos) == 0 {
		return errNoPositionForMarket(row.trader())
	}

	return errProfitAndLossValuesForTrader(pos, row)
}

func errProfitAndLossValuesForTrader(pos []*types.Position, row pnlRow) error {
	return formatDiff(
		fmt.Sprintf("invalid positions values for party(%v)", row.trader()),
		map[string]string{
			"volume":         strconv.FormatInt(row.volume(), 10),
			"unrealised PNL": strconv.FormatInt(row.unrealisedPNL(), 10),
			"realised PNL":   strconv.FormatInt(row.realisedPNL(), 10),
		},
		map[string]string{
			"volume":         strconv.FormatInt(pos[0].OpenVolume, 10),
			"unrealised PNL": strconv.FormatInt(pos[0].UnrealisedPnl, 10),
			"realised PNL":   strconv.FormatInt(pos[0].RealisedPnl, 10),
		},
	)
}

func errNoPositionForMarket(trader string) error {
	return fmt.Errorf("trader do not have a position, party(%v)", trader)
}

func areSamePosition(pos []*types.Position, row pnlRow) bool {
	return len(pos) == 1 &&
		pos[0].OpenVolume == row.volume() &&
		pos[0].RealisedPnl == row.realisedPNL() &&
		pos[0].UnrealisedPnl == row.unrealisedPNL()
}

func errCannotGetPositionForParty(trader string, err error) error {
	return fmt.Errorf("error getting party position, trader(%v), err(%v)", trader, err)
}

type pnlRow struct {
	row RowWrapper
}

func (r pnlRow) trader() string {
	return r.row.MustStr("trader")
}

func (r pnlRow) volume() int64 {
	return r.row.MustI64("volume")
}

func (r pnlRow) unrealisedPNL() int64 {
	return r.row.MustI64("unrealised pnl")
}

func (r pnlRow) realisedPNL() int64 {
	return r.row.MustI64("realised pnl")
}
