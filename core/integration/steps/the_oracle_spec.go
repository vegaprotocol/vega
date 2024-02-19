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
	"errors"
	"fmt"
	"time"

	dstypes "code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/integration/steps/market"
	"code.vegaprotocol.io/vega/libs/ptr"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	protoTypes "code.vegaprotocol.io/vega/protos/vega"
	datav1 "code.vegaprotocol.io/vega/protos/vega/data/v1"

	"github.com/cucumber/godog"
)

func TheOracleSpec(config *market.Config, name string, specType string, rawPubKeys string, table *godog.Table) error {
	pubKeys := StrSlice(rawPubKeys, ",")
	pubKeysSigners := make([]*datav1.Signer, len(pubKeys))
	for i, s := range pubKeys {
		pks := dstypes.CreateSignerFromString(s, dstypes.SignerTypePubKey)
		pubKeysSigners[i] = pks.IntoProto()
	}

	binding := &protoTypes.DataSourceSpecToFutureBinding{}

	rows := parseOracleSpecTable(table)
	filters := make([]*datav1.Filter, 0, len(rows))
	chainID := uint64(0)
	for _, r := range rows {
		row := oracleSpecRow{row: r}
		var numDec *uint64
		decimals, ok := row.propertyDecimals()
		if ok {
			numDec = ptr.From(decimals)
		}
		filter := &datav1.Filter{
			Key: &datav1.PropertyKey{
				Name:                row.propertyName(),
				Type:                row.propertyType(),
				NumberDecimalPlaces: numDec,
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
		if chainID == 0 {
			chainID = row.sourceChainID()
		}
	}
	dsSpec := &protoTypes.DataSourceSpec{
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
	}
	// set chain ID if provided
	if ex := dsSpec.Data.GetExternal(); ex != nil {
		if eo := ex.GetEthOracle(); eo != nil {
			eo.SourceChainId = chainID
		}
	}

	return config.OracleConfigs.Add(
		name,
		specType,
		dsSpec,
		binding,
	)
}

func parseOracleSpecTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"property",
		"type",
		"binding",
	}, []string{
		"decimals",
		"condition",
		"value",
		"source chain",
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

func (r oracleSpecRow) propertyDecimals() (uint64, bool) {
	return r.row.U64B("decimals")
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

func (r oracleSpecRow) sourceChainID() uint64 {
	if !r.row.HasColumn("source chain") {
		return 0
	}
	return r.row.MustU64("source chain")
}
