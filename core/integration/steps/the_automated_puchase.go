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
	"context"
	"time"

	"code.vegaprotocol.io/vega/core/datasource"
	"code.vegaprotocol.io/vega/core/integration/steps/market"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/cucumber/godog"
)

func TheAutomatedPurchasePrograms(
	config *market.Config,
	executionEngine Execution,
	table *godog.Table,
) error {
	rows := parseAutomatedPurchaseTable(table)
	aps := make(map[string]*types.NewProtocolAutomatedPurchaseChanges, len(rows))
	for _, row := range rows {
		r := apRow{row: row}
		ap, err := NewProtocolAutomatedPurchase(r, config)
		if err != nil {
			return err
		}
		aps[r.row.mustColumn("id")] = ap
	}
	for id, ap := range aps {
		if err := executionEngine.NewProtocolAutomatedPurchase(context.Background(), id, ap); err != nil {
			return err
		}
	}
	return nil
}

type apRow struct {
	row RowWrapper
}

func NewProtocolAutomatedPurchase(r apRow, config *market.Config) (*types.NewProtocolAutomatedPurchaseChanges, error) {
	duration, err := time.ParseDuration(r.row.MustStr("auction duration"))
	if err != nil {
		return nil, err
	}
	stalnessTol, err := time.ParseDuration(r.row.MustStr("price oracle staleness tolerance"))
	if err != nil {
		return nil, err
	}

	minSize, _ := num.UintFromString(r.row.MustStr("minimum auction size"), 10)
	maxSize, _ := num.UintFromString(r.row.MustStr("maximum auction size"), 10)
	oracleOffset, _ := num.DecimalFromString(r.row.MustStr("oracle offset factor"))

	auctionSchedule, _ := config.OracleConfigs.GetTimeTrigger(r.row.MustStr("auction schedule oracle"))
	auctionVolumeSnapshotSchedule, _ := config.OracleConfigs.GetTimeTrigger(r.row.MustStr("auction volume snapshot schedule oracle"))

	auctionPriceOracle, priceOracleBinding, _ := config.OracleConfigs.GetOracleDefinitionForCompositePrice(r.row.MustStr("price oracle"))
	expiry := r.row.MustI64("expiry timestamp")
	expiryTime := time.Unix(expiry, 0)

	priceOracle := datasource.FromOracleSpecProto(auctionPriceOracle)

	return &types.NewProtocolAutomatedPurchaseChanges{
		From:                          r.row.MustStr("from"),
		FromAccountType:               types.AccountType(vega.AccountType_value[r.row.MustStr("from account type")]),
		ToAccountType:                 types.AccountType(vega.AccountType_value[r.row.MustStr("to account type")]),
		MarketID:                      r.row.MustStr("market id"),
		AuctionDuration:               duration,
		MinimumAuctionSize:            minSize,
		MaximumAuctionSize:            maxSize,
		ExpiryTimestamp:               expiryTime,
		OraclePriceStalenessTolerance: stalnessTol,
		OracleOffsetFactor:            oracleOffset,
		AuctionSchedule:               auctionSchedule.GetDefinition(),
		AuctionVolumeSnapshotSchedule: auctionVolumeSnapshotSchedule.GetDefinition(),
		PriceOracle:                   priceOracle,
		PriceOracleBinding:            priceOracleBinding,
		AutomatedPurchaseSpecBinding: &vega.DataSourceSpecToAutomatedPurchaseBinding{
			AuctionScheduleProperty:               "vegaprotocol.builtin.timetrigger",
			AuctionVolumeSnapshotScheduleProperty: "vegaprotocol.builtin.timetrigger",
		},
	}, nil
}

func parseAutomatedPurchaseTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"id",
		"from",
		"from account type",
		"to account type",
		"market id",
		"price oracle",
		"price oracle staleness tolerance",
		"oracle offset factor",
		"auction schedule oracle",
		"auction volume snapshot schedule oracle",
		"auction duration",
		"minimum auction size",
		"maximum auction size",
		"expiry timestamp",
	}, []string{})
}
