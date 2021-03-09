package core_test

import (
	"fmt"
	"strings"

	"github.com/cucumber/godog/gherkin"
)

func ParseOracleDataPubKeys(rawPubKeys string) []string {
	return strings.Split(rawPubKeys, ",")
}

func ParseOracleDataProperties(table *gherkin.DataTable) (map[string]string, error) {
	properties := map[string]string{}

	for _, r := range table.Rows {
		row := propertyRow{row: r}
		if row.isHeader() {
			continue
		}
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
	row *gherkin.TableRow
}

func (r propertyRow) name() string {
	return val(r.row, 0)
}

func (r propertyRow) value() string {
	return val(r.row, 1)
}

func (r propertyRow) isHeader() bool {
	return r.name() == "name"
}
