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
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/core/integration/stubs"
	vtypes "code.vegaprotocol.io/vega/core/types"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/num"
	types "code.vegaprotocol.io/vega/protos/vega"
	"github.com/cucumber/godog"
)

func TheMarketStateShouldBeForMarket(
	engine Execution,
	market, expectedMarketStateStr string,
) error {
	expectedMarketState, err := MarketState(expectedMarketStateStr)
	panicW("market state", err)

	marketState, err := engine.GetMarketState(market)
	if err != nil {
		return errMarketDataNotFound(market, err)
	}

	if marketState != expectedMarketState {
		return errMismatchedMarketState(market, expectedMarketState, marketState)
	}
	return nil
}

func TheLastStateUpdateShouldBeForMarket(
	broker *stubs.BrokerStub,
	market, expectedMarketStateStr string,
) error {
	expectedMarketState, err := MarketState(expectedMarketStateStr)
	panicW("market state", err)

	lastMkt := broker.GetLastMarketUpdateState(market)
	if lastMkt == nil {
		return errMarketDataNotFound(market, fmt.Errorf("no market updates found"))
	}

	if lastMkt.State != expectedMarketState {
		return errMismatchedMarketState(market, expectedMarketState, lastMkt.State)
	}
	return nil
}

func TheMarketStateIsUpdatedTo(exec Execution, data *godog.Table) error {
	rows := parseStateUpdate(data)
	ctx := vgcontext.WithTraceID(context.Background(), "deadbeef")
	for _, r := range rows {
		mu := marketUpdateGov{
			row: r,
		}
		changes := &vtypes.MarketStateUpdateConfiguration{
			MarketID:        mu.MarketID(),
			SettlementPrice: mu.SettlementPrice(),
			UpdateType:      mu.MarketStateUpdate(),
		}
		if err := exec.UpdateMarketState(ctx, changes); err != nil {
			return err
		}
	}
	return nil
}

type marketUpdateGov struct {
	row RowWrapper
}

func parseStateUpdate(data *godog.Table) []RowWrapper {
	return StrictParseTable(data, []string{
		"market id",
		"state",
		"settlement price",
	}, nil)
}

func errMismatchedMarketState(market string, expectedMarketState, marketState types.Market_State) error {
	return formatDiff(
		fmt.Sprintf("unexpected market state for market \"%s\"", market),
		map[string]string{
			"market state": expectedMarketState.String(),
		},
		map[string]string{
			"market state": marketState.String(),
		},
	)
}

func (m marketUpdateGov) MarketID() string {
	return m.row.MustStr("market id")
}

func (m marketUpdateGov) MarketStateUpdate() vtypes.MarketStateUpdateType {
	return m.row.MustMarketUpdateState("state")
}

func (m marketUpdateGov) SettlementPrice() *num.Uint {
	return m.row.MustUint("settlement price")
}
