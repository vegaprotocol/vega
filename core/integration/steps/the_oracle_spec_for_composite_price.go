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
	dstypes "code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/integration/steps/market"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	protoTypes "code.vegaprotocol.io/vega/protos/vega"
	datav1 "code.vegaprotocol.io/vega/protos/vega/data/v1"

	"github.com/cucumber/godog"
)

func TheCompositePriceOracleSpec(config *market.Config, keys string, table *godog.Table) error {
	pubKeys := StrSlice(keys, ",")
	pubKeysSigners := make([]*datav1.Signer, len(pubKeys))
	for i, s := range pubKeys {
		pks := dstypes.CreateSignerFromString(s, dstypes.SignerTypePubKey)
		pubKeysSigners[i] = pks.IntoProto()
	}

	rows := parseCompositePriceOracleTable(table)
	for _, r := range rows {
		row := compositePriceOracleRow{row: r}
		name := row.name()
		priceP := row.priceProperty()
		binding := &protoTypes.SpecBindingForCompositePrice{
			PriceSourceProperty: priceP,
		}
		filters := []*datav1.Filter{
			{
				Key: &datav1.PropertyKey{
					Name:                priceP,
					Type:                row.priceType(),
					NumberDecimalPlaces: row.priceDecimals(),
				},
				Conditions: []*datav1.Condition{},
			},
		}

		ds := &protoTypes.DataSourceSpec{
			Id: vgrand.RandomStr(10),
			Data: protoTypes.NewDataSourceDefinition(
				protoTypes.DataSourceContentTypeOracle,
			).SetOracleConfig(
				&protoTypes.DataSourceDefinitionExternal_Oracle{
					Oracle: &protoTypes.DataSourceSpecConfiguration{
						Signers: pubKeysSigners,
						Filters: filters[:1],
					},
				},
			),
		}
		config.OracleConfigs.AddCompositePriceOracle(name, ds, binding)
	}
	return nil
}

func parseCompositePriceOracleTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"name",
		"price property",
		"price type",
	}, []string{
		"price decimals",
	})
}

type compositePriceOracleRow struct {
	row RowWrapper
}

func (p compositePriceOracleRow) name() string {
	return p.row.MustStr("name")
}

func (p compositePriceOracleRow) priceProperty() string {
	return p.row.MustStr("price property")
}

func (p compositePriceOracleRow) priceType() datav1.PropertyKey_Type {
	return p.row.MustOracleSpecPropertyType("price type")
}

func (p compositePriceOracleRow) priceDecimals() *uint64 {
	if !p.row.HasColumn("price decimals") {
		return nil
	}
	v := p.row.MustU64("price decimals")
	return &v
}
