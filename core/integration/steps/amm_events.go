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

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"
	"github.com/cucumber/godog"
)

func AMMPoolStatusShouldBe(broker *stubs.BrokerStub, table *godog.Table) error {
	recent := broker.GetLastAMMPoolEvents()
	for _, r := range parseAMMEventTable(table) {
		row := ammEvtRow{
			r: r,
		}
		mID, pID := row.market(), row.party()
		mmap, ok := recent[mID]
		if !ok {
			return fmt.Errorf("no AMM events found for market %s", mID)
		}
		pEvt, ok := mmap[pID]
		if !ok {
			return fmt.Errorf("no AMM events found for party %s in market %s", pID, mID)
		}
		if err := row.matchesEvt(pEvt); err != nil {
			return err
		}
	}
	return nil
}

func ExpectToSeeAMMEvents(broker *stubs.BrokerStub, table *godog.Table) error {
	evtMap := broker.GetAMMPoolEventMap()
	for _, r := range parseAMMEventTable(table) {
		row := ammEvtRow{
			r: r,
		}
		mID, pID := row.market(), row.party()
		mmap, ok := evtMap[mID]
		if !ok {
			return fmt.Errorf("no AMM events found for market %s", mID)
		}
		pEvts, ok := mmap[pID]
		if !ok {
			return fmt.Errorf("no AMM events found for party %s in market %s", pID, mID)
		}
		var err error
		for _, e := range pEvts {
			if err = row.matchesEvt(e); err == nil {
				break
			}
		}
		if err != nil {
			return fmt.Errorf("expected AMM event for party %s on market %s not found, last AMM pool event mismatch: %v", pID, mID, err)
		}
	}
	return nil
}

type ammEvtRow struct {
	r RowWrapper
}

func parseAMMEventTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"market id",
		"amount",
		"status",
	}, []string{
		"reason",
		"base",
		"lower bound",
		"upper bound",
		"lower margin ratio",
		"upper margin ratio",
	})
}

func DebugAMMPoolEvents(broker *stubs.BrokerStub, log *logging.Logger) error {
	evts := broker.GetAMMPoolEvents()
	logEvents(log, evts)
	return nil
}

func DebugAMMPoolEventsForPartyMarket(broker *stubs.BrokerStub, log *logging.Logger, party, market *string) error {
	if party == nil && market == nil {
		return DebugAMMPoolEvents(broker, log)
	}
	if market == nil {
		logEvents(log, broker.GetAMMPoolEventsByParty(*party))
		return nil
	}
	if party == nil {
		logEvents(log, broker.GetAMMPoolEventsByMarket(*market))
		return nil
	}
	logEvents(log, broker.GetAMMPoolEventsByPartyAndMarket(*party, *market))
	return nil
}

func logEvents(log *logging.Logger, evts []*events.AMMPool) {
	for _, e := range evts {
		pool := e.AMMPool()
		if pool.Parameters == nil {
			log.Info(fmt.Sprintf("AMM Party: %s on Market: %s - Amount: %s - no parameters", pool.PartyId, pool.MarketId, pool.Commitment))
			continue
		}
		log.Info(fmt.Sprintf(
			"AMM Party: %s on Market: %s - Amount: %s\nStatus: %s, Reason: %s\n Base: %s, Bounds: %s-%s, Margin ratios: %s-%s",
			pool.PartyId, pool.MarketId, pool.Commitment,
			pool.Status.String(), pool.StatusReason.String(),
			pool.Parameters.Base, pool.Parameters.LowerBound, pool.Parameters.UpperBound,
			pool.Parameters.MarginRatioAtLowerBound, pool.Parameters.MarginRatioAtUpperBound,
		))
	}
}

func (a ammEvtRow) matchesEvt(e *events.AMMPool) error {
	pool := e.AMMPool()

	if pool.PartyId != a.party() || pool.MarketId != a.market() || pool.Commitment != a.r.MustStr("amount") || pool.Status != a.status() {
		return fmt.Errorf(
			"expected party %s, market %s, amount %s, status %s - instead got %s, %s, %s, %s",
			a.party(), a.market(), a.r.MustStr("amount"), a.status().String(),
			pool.PartyId, pool.MarketId, pool.Commitment, pool.Status.String(),
		)
	}
	got := make([]any, 0, 10)
	got = append(got, pool.PartyId, pool.MarketId, pool.Commitment, pool.Status.String())
	eFmt := "mismatch for %s, %s, %s, %s"
	if psr, check := a.reason(); check {
		if pool.StatusReason != psr {
			got = append(got, psr.String, pool.StatusReason.String())
			return fmt.Errorf(eFmt+" expected reason %s - instead got %s", got...)
		}
		got = append(got, psr.String())
		eFmt = eFmt + ", %s"
	}
	checks := map[string]string{
		"base":               pool.Parameters.Base,
		"lower bound":        pool.Parameters.LowerBound,
		"upper bound":        pool.Parameters.UpperBound,
		"lower margin ratio": pool.Parameters.MarginRatioAtLowerBound,
		"upper margin ratio": pool.Parameters.MarginRatioAtUpperBound,
	}
	for name, val := range checks {
		if !a.r.HasColumn(name) {
			continue
		}
		if exp := a.r.MustStr(name); val != exp {
			got = append(got, name, exp, val)
			return fmt.Errorf(eFmt+" expected %s %s - instead got %s", got...)
		}
		got = append(got, val)
		eFmt = eFmt + ", %s"
	}
	return nil
}

func (a ammEvtRow) party() string {
	return a.r.MustStr("party")
}

func (a ammEvtRow) market() string {
	return a.r.MustStr("market id")
}

func (a ammEvtRow) status() types.AMMPoolStatus {
	return a.r.MustAMMPoolStatus("status")
}

func (a ammEvtRow) reason() (types.AMMPoolStatusReason, bool) {
	if !a.r.HasColumn("reason") {
		return types.AMMPoolStatusReasonUnspecified, false
	}
	sr := a.r.MustPoolStatusReason("reason")
	return sr, true
}
