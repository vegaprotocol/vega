package steps

import (
	dstypes "code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/integration/steps/market"
	"code.vegaprotocol.io/vega/libs/num"
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
		binding := &protoTypes.DataSourceSpecToPerpsBinding{
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
	}
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
		"clam upper bound",
		"quote name",
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
	return r.row.MustOracleSpecPropertyType("settlement type")
}

func (p perpOracleRow) ScheduleProperty() string {
	return p.row.MustStr("schedule property")
}

func (p perpOracleRow) ScheduleType() datav1.PropertyKey_Type {
	return r.row.MustOracleSpecPropertyType("schedule type")
}

func (p perpOracleRow) QuoteName() string {
	if !p.row.HasColumn("quote name") {
		return ""
	}
	return p.row.MustStr("quote name")
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
