package steps

import (
	"github.com/cucumber/godog"

	"code.vegaprotocol.io/vega/core/integration/steps/market"
	types "code.vegaprotocol.io/vega/protos/vega"
)

func TheLiquiditySLAPArams(config *market.Config, name string, table *godog.Table) error {
	row := slaParamRow{row: parseSLAParamsTable(table)[0]}

	return config.LiquiditySLAParams.Add(
		name,
		&types.LiquiditySLAParameters{
			PriceRange:                      row.priceRange(),
			CommitmentMinTimeFraction:       row.commitmentMinTimeFraction(),
			SlaCompetitionFactor:            row.slaCompetitionFactor(),
			ProvidersFeeCalculationTimeStep: row.providersFeeCalculationTimeStep(),
			PerformanceHysteresisEpochs:     uint64(row.performanceHysteresisEpochs()),
		})
}

func parseSLAParamsTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"price range",
		"commitment min time fraction",
		"providers fee calculation time step",
		"performance hysteresis epochs",
		"sla competition factor",
	}, []string{})
}

type slaParamRow struct {
	row RowWrapper
}

func (r slaParamRow) priceRange() string {
	return r.row.MustStr("price range")
}

func (r slaParamRow) commitmentMinTimeFraction() string {
	return r.row.MustStr("commitment min time fraction")
}

func (r slaParamRow) providersFeeCalculationTimeStep() int64 {
	return r.row.MustI64("providers fee calculation time step")
}

func (r slaParamRow) performanceHysteresisEpochs() int64 {
	return r.row.MustI64("performance hysteresis epochs")
}

func (r slaParamRow) slaCompetitionFactor() string {
	return r.row.MustStr("sla competition factor")
}
