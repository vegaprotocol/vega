package steps

import (
	"fmt"

	"github.com/cucumber/godog"

	types "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/integration/steps/market"
)

func ThePriceMonitoring(config *market.Config, name string, rawUpdateFrequency string, table *godog.Table) error {
	updateFrequency, err := I64(rawUpdateFrequency)
	if err != nil {
		panicW("update frequency", err)
	}

	rows := parsePriceMonitoringTable(table)
	triggers := make([]*types.PriceMonitoringTrigger, 0, len(rows))
	for _, r := range rows {
		row := priceMonitoringRow{row: r}
		p := &types.PriceMonitoringTrigger{
			Horizon:          row.horizon(),
			Probability:      fmt.Sprintf("%0.16f", row.probability()),
			AuctionExtension: row.auctionExtension(),
		}
		triggers = append(triggers, p)
	}

	return config.PriceMonitoring.Add(
		name,
		&types.PriceMonitoringSettings{
			Parameters: &types.PriceMonitoringParameters{
				Triggers: triggers,
			},
			UpdateFrequency: updateFrequency,
		},
	)
}

func parsePriceMonitoringTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"horizon",
		"probability",
		"auction extension",
	}, []string{})
}

type priceMonitoringRow struct {
	row RowWrapper
}

func (r priceMonitoringRow) horizon() int64 {
	return r.row.MustI64("horizon")
}

func (r priceMonitoringRow) probability() float64 {
	return r.row.MustF64("probability")
}

func (r priceMonitoringRow) auctionExtension() int64 {
	return r.row.MustI64("auction extension")
}
