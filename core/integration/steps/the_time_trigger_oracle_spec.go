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

	"code.vegaprotocol.io/vega/core/integration/steps/market"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	protoTypes "code.vegaprotocol.io/vega/protos/vega"
	datav1 "code.vegaprotocol.io/vega/protos/vega/data/v1"

	"github.com/cucumber/godog"
)

func TheTimeTriggerOracleSpec(config *market.Config, table *godog.Table) error {
	rows := parseTimeTriggerSpecTable(table)
	for _, r := range rows {
		row := internalTimeTriggerOracleSpecRow{row: r}
		initial := row.initial()
		name := row.name()
		spec := &protoTypes.DataSourceSpec{
			Id: vgrand.RandomStr(10),
			Data: &protoTypes.DataSourceDefinition{
				SourceType: &protoTypes.DataSourceDefinition_Internal{
					Internal: &protoTypes.DataSourceDefinitionInternal{
						SourceType: &protoTypes.DataSourceDefinitionInternal_TimeTrigger{
							TimeTrigger: &protoTypes.DataSourceSpecConfigurationTimeTrigger{
								Triggers: []*datav1.InternalTimeTrigger{
									{
										Initial: &initial,
										Every:   row.every(),
									},
								},
								Conditions: []*datav1.Condition{
									{
										Operator: datav1.Condition_OPERATOR_GREATER_THAN_OR_EQUAL,
										Value:    fmt.Sprintf("%d", 0),
									},
								},
							},
						},
					},
				},
			},
		}
		config.OracleConfigs.AddTimeTrigger(name, spec)
	}
	return nil
}

func parseTimeTriggerSpecTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"name",
		"initial",
		"every",
	}, []string{})
}

type internalTimeTriggerOracleSpecRow struct {
	row RowWrapper
}

func (r internalTimeTriggerOracleSpecRow) name() string {
	return r.row.MustStr("name")
}

func (r internalTimeTriggerOracleSpecRow) initial() int64 {
	return r.row.MustI64("initial")
}

func (r internalTimeTriggerOracleSpecRow) every() int64 {
	return r.row.MustI64("every")
}
