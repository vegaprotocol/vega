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
	"errors"
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
			MarketID:   mu.MarketID(),
			UpdateType: mu.MarketStateUpdate(),
		}
		if r.HasColumn("settlement price") {
			changes.SettlementPrice = mu.SettlementPrice()
		}
		expErr := mu.Err()
		if err := exec.UpdateMarketState(ctx, changes); err != nil {
			if expErr != nil && err.Error() != expErr.Error() {
				return err
			}
		} else if expErr != nil {
			return fmt.Errorf("expected error %s, instead got no error", expErr.Error())
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
	}, []string{
		"settlement price",
		"error",
	})
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

func (m marketUpdateGov) Err() error {
	if m.row.HasColumn("error") {
		return errors.New(m.row.MustStr("error"))
	}
	return nil
}
