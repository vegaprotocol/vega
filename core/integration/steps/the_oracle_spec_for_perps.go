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
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	protoTypes "code.vegaprotocol.io/vega/protos/vega"
	datav1 "code.vegaprotocol.io/vega/protos/vega/data/v1"

	"github.com/cucumber/godog"
)

func ThePerpsOracleSpec(config *market.Config, keys string, table *godog.Table) error {
	pubKeys := StrSlice(keys, ",")
	pubKeysSigners := make([]*datav1.Signer, len(pubKeys))
	for i, s := range pubKeys {
		pks := dstypes.CreateSignerFromString(s, dstypes.SignerTypePubKey)
		pubKeysSigners[i] = pks.IntoProto()
	}

	rows := parseOraclePerpsTable(table)
	for _, r := range rows {
		row := perpOracleRow{row: r}
		name := row.Name()
		settleP, scheduleP := row.SettlementProperty(), row.ScheduleProperty()
		binding := &protoTypes.DataSourceSpecToPerpetualBinding{
			SettlementDataProperty:     settleP,
			SettlementScheduleProperty: scheduleP,
		}
		filters := []*datav1.Filter{
			{
				Key: &datav1.PropertyKey{
					Name:                settleP,
					Type:                row.SettlementType(),
					NumberDecimalPlaces: row.SettlementDecimals(),
				},
				Conditions: []*datav1.Condition{},
			},
			{
				Key: &datav1.PropertyKey{
					Name: scheduleP,
					Type: row.ScheduleType(),
				},
				Conditions: []*datav1.Condition{},
			},
		}
		perp := &protoTypes.Perpetual{
			SettlementAsset:          row.Asset(),
			QuoteName:                row.QuoteName(),
			MarginFundingFactor:      row.MarginFundingFactor().String(),
			InterestRate:             row.InterestRate().String(),
			ClampLowerBound:          row.LowerClamp().String(),
			ClampUpperBound:          row.UpperClamp().String(),
			FundingRateScalingFactor: row.FundingRateScalingFactor(),
			FundingRateLowerBound:    row.FundingRateLowerBound(),
			FundingRateUpperBound:    row.FundingRateUpperBound(),
			DataSourceSpecForSettlementData: &protoTypes.DataSourceSpec{
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
			},
			DataSourceSpecForSettlementSchedule: &protoTypes.DataSourceSpec{
				Id: vgrand.RandomStr(10),
				Data: protoTypes.NewDataSourceDefinition(
					protoTypes.DataSourceContentTypeOracle,
				).SetOracleConfig(
					&protoTypes.DataSourceDefinitionExternal_Oracle{
						Oracle: &protoTypes.DataSourceSpecConfiguration{
							Signers: pubKeysSigners,
							Filters: filters[1:],
						},
					},
				),
			},
			DataSourceSpecBinding: binding,
		}
		if err := config.OracleConfigs.AddPerp(name, perp); err != nil {
			return err
		}
	}
	return nil
}

func parseOraclePerpsTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"name",
		"asset",
		"settlement property",
		"settlement type",
		"schedule property",
		"schedule type",
	}, []string{
		"settlement decimals",
		"margin funding factor",
		"interest rate",
		"clamp lower bound",
		"clamp upper bound",
		"quote name",
		"funding rate scaling factor",
		"funding rate lower bound",
		"funding rate upper bound",
	})
}

type perpOracleRow struct {
	row RowWrapper
}

func (p perpOracleRow) Name() string {
	return p.row.MustStr("name")
}

func (p perpOracleRow) Asset() string {
	return p.row.MustStr("asset")
}

func (p perpOracleRow) SettlementProperty() string {
	return p.row.MustStr("settlement property")
}

func (p perpOracleRow) SettlementType() datav1.PropertyKey_Type {
	return p.row.MustOracleSpecPropertyType("settlement type")
}

func (p perpOracleRow) ScheduleProperty() string {
	return p.row.MustStr("schedule property")
}

func (p perpOracleRow) ScheduleType() datav1.PropertyKey_Type {
	return p.row.MustOracleSpecPropertyType("schedule type")
}

func (p perpOracleRow) QuoteName() string {
	if !p.row.HasColumn("quote name") {
		return ""
	}
	return p.row.MustStr("quote name")
}

func (p perpOracleRow) FundingRateScalingFactor() *string {
	if !p.row.HasColumn("funding rate scaling factor") {
		return nil
	}
	return ptr.From(p.row.MustDecimal("funding rate scaling factor").String())
}

func (p perpOracleRow) FundingRateUpperBound() *string {
	if !p.row.HasColumn("funding rate upper bound") {
		return nil
	}
	return ptr.From(p.row.MustDecimal("funding rate upper bound").String())
}

func (p perpOracleRow) FundingRateLowerBound() *string {
	if !p.row.HasColumn("funding rate lower bound") {
		return nil
	}
	return ptr.From(p.row.MustDecimal("funding rate lower bound").String())
}

func (p perpOracleRow) MarginFundingFactor() num.Decimal {
	if !p.row.HasColumn("margin funding factor") {
		return num.DecimalZero()
	}
	return p.row.MustDecimal("margin funding factor")
}

func (p perpOracleRow) LowerClamp() num.Decimal {
	if !p.row.HasColumn("clamp lower bound") {
		return num.DecimalZero()
	}
	return p.row.MustDecimal("clamp lower bound")
}

func (p perpOracleRow) UpperClamp() num.Decimal {
	if !p.row.HasColumn("clamp upper bound") {
		return num.DecimalZero()
	}
	return p.row.MustDecimal("clamp upper bound")
}

func (p perpOracleRow) InterestRate() num.Decimal {
	if !p.row.HasColumn("interest rate") {
		return num.DecimalZero()
	}
	return p.row.MustDecimal("interest rate")
}

func (p perpOracleRow) SettlementDecimals() *uint64 {
	if !p.row.HasColumn("settlement decimals") {
		return nil
	}
	v := p.row.MustU64("settlement decimals")
	return &v
}
