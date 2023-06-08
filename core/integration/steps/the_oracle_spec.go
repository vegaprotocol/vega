// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package steps

import (
	"errors"
	"fmt"
	"time"

	"github.com/cucumber/godog"

	"code.vegaprotocol.io/vega/core/integration/steps/market"
	"code.vegaprotocol.io/vega/core/types"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	protoTypes "code.vegaprotocol.io/vega/protos/vega"
	datav1 "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

func TheOracleSpec(config *market.Config, name string, specType string, rawPubKeys string, table *godog.Table) error {
	pubKeys := StrSlice(rawPubKeys, ",")
	pubKeysSigners := make([]*datav1.Signer, len(pubKeys))
	for i, s := range pubKeys {
		pks := types.CreateSignerFromString(s, types.DataSignerTypePubKey)
		pubKeysSigners[i] = pks.IntoProto()
	}

	binding := &protoTypes.DataSourceSpecToFutureBinding{}

	rows := parseOracleSpecTable(table)
	filters := make([]*datav1.Filter, 0, len(rows))
	for _, r := range rows {
		row := oracleSpecRow{row: r}
		filter := &datav1.Filter{
			Key: &datav1.PropertyKey{
				Name: row.propertyName(),
				Type: row.propertyType(),
			},
			Conditions: []*datav1.Condition{},
		}

		if r.HasColumn("condition") != r.HasColumn("value") {
			return errors.New("condition and value columns require each other")
		}

		if r.HasColumn("condition") {
			value := row.value()

			if row.propertyType() == datav1.PropertyKey_TYPE_TIMESTAMP {
				expiry, err := time.Parse(time.RFC3339, value)
				if err != nil {
					panic(fmt.Errorf("cannot parse expiry condition: %w", err))
				}
				value = fmt.Sprintf("%d", expiry.Unix())
			}

			filter.Conditions = append(filter.Conditions,
				&datav1.Condition{
					Operator: row.condition(),
					Value:    value,
				},
			)
		}

		filters = append(filters, filter)

		if row.destination() == "settlement data" {
			binding.SettlementDataProperty = row.propertyName()
		}
		if row.destination() == "trading termination" {
			binding.TradingTerminationProperty = row.propertyName()
		}
	}

	return config.OracleConfigs.Add(
		name,
		specType,
		&protoTypes.DataSourceSpec{
			Id: vgrand.RandomStr(10),
			Data: protoTypes.NewDataSourceDefinition(
				protoTypes.DataSourceContentTypeOracle,
			).SetOracleConfig(
				&protoTypes.DataSourceDefinitionExternal_Oracle{
					Oracle: &protoTypes.DataSourceSpecConfiguration{
						Signers: pubKeysSigners,
						Filters: filters,
					},
				},
			),
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
		"value",
	})
}

type oracleSpecRow struct {
	row RowWrapper
}

func (r oracleSpecRow) propertyName() string {
	return r.row.MustStr("property")
}

func (r oracleSpecRow) propertyType() datav1.PropertyKey_Type {
	return r.row.MustOracleSpecPropertyType("type")
}

func (r oracleSpecRow) destination() string {
	return r.row.MustStr("binding")
}

func (r oracleSpecRow) condition() datav1.Condition_Operator {
	return r.row.MustOracleSpecConditionOperator("condition")
}

func (r oracleSpecRow) value() string {
	return r.row.MustStr("value")
}
