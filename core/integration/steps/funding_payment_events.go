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
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/logging"

	"github.com/cucumber/godog"
)

func TheFollowingFundingPeriodEventsShouldBeEmitted(broker *stubs.BrokerStub, table *godog.Table) error {
	fundingPeriodEvents := broker.GetFundingPeriodEvents()
	for _, row := range parseFundingPeriodEventTable(table) {
		fpe := FundingPeriodEventWrapper{
			row: row,
		}

		matched := false
		for _, evt := range fundingPeriodEvents {
			if checkFundingPeriodEvent(evt, fpe) {
				matched = true
				break
			}
		}
		if !matched {
			start, _ := fpe.Start()
			end, _ := fpe.End()
			return fmt.Errorf("Funding period event with start=%v, end=%v, internal TWAP=%s, external TWAP=%s not found", start, end, fpe.InternalTWAP(), fpe.ExternalTWAP())
		}
	}
	return nil
}

func DebugFundingPeriodEventss(broker *stubs.BrokerStub, log *logging.Logger) {
	log.Info("DUMPING FUNDING PERIOD EVENTS")
	data := broker.GetFundingPeriodEvents()
	for _, evt := range data {
		p := evt.Proto()
		log.Infof("%s\n", p.String())
	}
}

func VerifyTime(now time.Time, expected int64) error {
	actual := now.Unix()
	if actual == expected || now.UnixNano() == expected {
		return nil
	}
	return fmt.Errorf("Expected unix time=%v, actual=%v", expected, actual)
}

func checkFundingPeriodEvent(evt events.FundingPeriod, row FundingPeriodEventWrapper) bool {
	fundingPeriod := evt.FundingPeriod()

	expectedStart, b := row.Start()
	actualStart := fundingPeriod.GetStart()
	if b && !actualEqualsExpectedInSecondsOrNanos(actualStart, expectedStart) {
		return false
	}
	expectedEnd, b := row.End()
	actualEnd := fundingPeriod.GetEnd()
	if b && !actualEqualsExpectedInSecondsOrNanos(actualEnd, expectedEnd) {
		return false
	}
	expectedInternalTwap := row.InternalTWAP()
	actualInternalTwap := fundingPeriod.GetInternalTwap()
	if actualInternalTwap == "" && len(expectedInternalTwap) > 0 || expectedInternalTwap != actualInternalTwap {
		return false
	}
	expectedExternalTwap := row.ExternalTWAP()
	actualExternalTwap := fundingPeriod.GetExternalTwap()

	if actualExternalTwap == "" && len(expectedExternalTwap) > 0 || expectedExternalTwap != actualExternalTwap {
		return false
	}
	expectedFundingPayment, b := row.FundingPayment()
	actualFundingPayment := fundingPeriod.GetFundingPayment()
	if b && (actualFundingPayment == "" && len(expectedFundingPayment) > 0 || expectedFundingPayment != actualFundingPayment) {
		return false
	}
	expectedFundingRate, b := row.FundingRate()
	actualFundingRate := fundingPeriod.GetFundingRate()
	if b && (actualFundingRate == "" && len(expectedFundingRate) > 0 || expectedFundingRate != actualFundingRate) {
		return false
	}

	return true
}

func actualEqualsExpectedInSecondsOrNanos(actual, expected int64) bool {
	return expected == actual || expected*int64(time.Second) == actual
}

type FundingPeriodEventWrapper struct {
	row RowWrapper
}

func parseFundingPeriodEventTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"internal twap",
		"external twap",
	}, []string{
		"funding payment",
		"funding rate",
		"start",
		"end",
	})
}

func (f FundingPeriodEventWrapper) InternalTWAP() string {
	return f.row.MustStr("internal twap")
}

func (f FundingPeriodEventWrapper) ExternalTWAP() string {
	return f.row.MustStr("external twap")
}

func (f FundingPeriodEventWrapper) FundingPayment() (string, bool) {
	return f.row.StrB("funding payment")
}

func (f FundingPeriodEventWrapper) FundingRate() (string, bool) {
	return f.row.StrB("funding rate")
}

func (f FundingPeriodEventWrapper) Start() (int64, bool) {
	return f.row.I64B("start")
}

func (f FundingPeriodEventWrapper) End() (int64, bool) {
	return f.row.I64B("end")
}
