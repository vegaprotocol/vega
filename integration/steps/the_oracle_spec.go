package steps

import (
	"github.com/cucumber/godog/gherkin"

	"code.vegaprotocol.io/vega/integration/steps/market"
	types "code.vegaprotocol.io/vega/proto"
	oraclesv1 "code.vegaprotocol.io/vega/proto/oracles/v1"
)

func TheOracleSpec(config *market.Config, name string, rawPubKeys string, table *gherkin.DataTable) error {
	pubKeys := StrSlice(rawPubKeys, ",")

	binding := &types.OracleSpecToFutureBinding{}

	var filters []*oraclesv1.Filter
	for _, r := range parseOracleSpecTable(table) {
		row := oracleSpecRow{row: r}
		filter := &oraclesv1.Filter{
			Key: &oraclesv1.PropertyKey{
				Name: row.propertyName(),
				Type: row.propertyType(),
			},
			Conditions: []*oraclesv1.Condition{},
		}
		filters = append(filters, filter)

		if row.destination() == "settlement price" {
			binding.SettlementPriceProperty = row.propertyName()
		}
	}

	return config.OracleConfigs.Add(
		name,
		&oraclesv1.OracleSpec{
			PubKeys: pubKeys,
			Filters: filters,
		},
		binding,
	)
}

func parseOracleSpecTable(table *gherkin.DataTable) []RowWrapper {
	return TableWrapper(*table).StrictParse([]string{
		"property",
		"type",
		"binding",
	}, []string{})
}

type oracleSpecRow struct {
	row RowWrapper
}

func (r oracleSpecRow) propertyName() string {
	return r.row.MustStr("property")
}

func (r oracleSpecRow) propertyType() oraclesv1.PropertyKey_Type {
	return r.row.MustOracleSpecPropertyType("type")
}

func (r oracleSpecRow) destination() string {
	return r.row.MustStr("binding")
}
