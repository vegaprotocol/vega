package steps

import (
	"fmt"
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
	return fmt.Errorf("invalid positions api values for party(%v): volume (expected %v, got %v), unrealisedPNL (expected %v, got %v), realisedPNL (expected %v, got %v)",
		row.trader(), row.volume(), pos[0].OpenVolume, row.unrealisedPNL(), pos[0].UnrealisedPnl, row.realisedPNL(), pos[0].RealisedPnl)
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
