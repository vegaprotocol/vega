package steps

import (
	"context"
	"fmt"
	"strings"

	"github.com/cucumber/godog/gherkin"

	"code.vegaprotocol.io/vega/oracles"
)

func OraclesBroadcastDataSignedWithKeys(
	oracleEngine *oracles.Engine,
	rawPubKeys string,
	rawProperties *gherkin.DataTable,
) error {
	pubKeys := parseOracleDataPubKeys(rawPubKeys)

	properties, err := parseOracleDataProperties(rawProperties)
	if err != nil {
		return err
	}

	return oracleEngine.BroadcastData(context.Background(), oracles.OracleData{
		PubKeys: pubKeys,
		Data:    properties,
	})
}

func parseOracleDataPubKeys(rawPubKeys string) []string {
	if len(rawPubKeys) == 0 {
		return nil
	}
	return strings.Split(rawPubKeys, ",")
}

func parseOracleDataProperties(table *gherkin.DataTable) (map[string]string, error) {
	properties := map[string]string{}

	for _, r := range TableWrapper(*table).Parse() {
		row := propertyRow{row: r}
		_, ok := properties[row.name()]
		if ok {
			return nil, errPropertyRedeclared(row.name())
		}
		properties[row.name()] = row.value()
	}

	return properties, nil
}
func errPropertyRedeclared(name string) error {
	return fmt.Errorf("property %s has been declared multiple times", name)
}

// propertyRow wraps the declaration of the properties of an oracle data
type propertyRow struct {
	row RowWrapper
}

func (r propertyRow) name() string {
	return r.row.Str("name")
}

func (r propertyRow) value() string {
	return r.row.Str("value")
}
