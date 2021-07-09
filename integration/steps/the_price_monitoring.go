package steps

import (
	"github.com/cucumber/godog/gherkin"

	"code.vegaprotocol.io/vega/integration/steps/market"
	types "code.vegaprotocol.io/vega/proto"
)

func ThePriceMonitoring(config *market.Config, name string, rawUpdateFrequency string, table *gherkin.DataTable) error {
	updateFrequency, err := I64(rawUpdateFrequency)
	if err != nil {
		panicW("update frequency", err)
	}

	var triggers []*types.PriceMonitoringTrigger
	for _, r := range parsePriceMonitoringTable(table) {
		row := priceMonitoringRow{row: r}
		p := &types.PriceMonitoringTrigger{
			Horizon:          row.horizon(),
			Probability:      row.probability(),
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

func parsePriceMonitoringTable(table *gherkin.DataTable) []RowWrapper {
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
