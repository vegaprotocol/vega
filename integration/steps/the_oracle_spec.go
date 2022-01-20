package steps

import (
	"fmt"
	"time"

	"github.com/cucumber/godog"

	types "code.vegaprotocol.io/protos/vega"
	oraclesv1 "code.vegaprotocol.io/protos/vega/oracles/v1"
	vgrand "code.vegaprotocol.io/shared/libs/rand"
	"code.vegaprotocol.io/vega/integration/steps/market"
)

func TheOracleSpec(config *market.Config, name string, specType string, rawPubKeys string, table *godog.Table) error {
	pubKeys := StrSlice(rawPubKeys, ",")
	binding := &types.OracleSpecToFutureBinding{}

	rows := parseOracleSpecTable(table)
	filters := make([]*oraclesv1.Filter, 0, len(rows))
	for _, r := range rows {
		row := oracleSpecRow{row: r}
		filter := &oraclesv1.Filter{
			Key: &oraclesv1.PropertyKey{
				Name: row.propertyName(),
				Type: row.propertyType(),
			},
			Conditions: []*oraclesv1.Condition{},
		}

		if condition := row.condition(); condition != "" {
			expiry, err := time.Parse(time.RFC3339, condition)
			if err != nil {
				panic(fmt.Errorf("cannot parse expiry condition: %w", err))
			}

			filter.Conditions = append(filter.Conditions,
				&oraclesv1.Condition{
					Operator: oraclesv1.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
					Value:    fmt.Sprintf("%d", expiry.UnixNano()),
				},
			)
		}

		filters = append(filters, filter)

		if row.destination() == "settlement price" {
			binding.SettlementPriceProperty = row.propertyName()
		}
		if row.destination() == "trading termination" {
			binding.TradingTerminationProperty = row.propertyName()
		}
	}

	return config.OracleConfigs.Add(
		name,
		specType,
		&oraclesv1.OracleSpec{
			Id:      vgrand.RandomStr(10),
			PubKeys: pubKeys,
			Filters: filters,
		},
		binding,
	)
}

func parseOracleSpecTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"property",
		"type",
		"binding",
	}, []string{
		"condition",
	})
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

func (r oracleSpecRow) condition() string {
	if !r.row.HasColumn("condition") {
		return ""
	}

	return r.row.Str("condition")
}
