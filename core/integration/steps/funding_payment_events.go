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
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

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

func TheFollowingFundingPaymentEventsShouldBeEmitted(broker *stubs.BrokerStub, table *godog.Table) error {
	paymentEvts := broker.GetFundginPaymentEvents()
	checkLoss := false
	rows := parseFundingPaymentsTable(table)
	matchers := make([]FundingPaymentsWrapper, 0, len(rows))
	for _, row := range rows {
		w := FundingPaymentsWrapper{
			r: row,
		}
		matchers = append(matchers, w)
		checkLoss = (checkLoss || w.CheckLoss())
	}
	// map the events by party and market
	lsEvt := map[string]map[string][]*events.LossSoc{}
	if checkLoss {
		for _, ls := range broker.GetLossSoc() {
			mID, pID := ls.MarketID(), ls.PartyID()
			mmap, ok := lsEvt[mID]
			if !ok {
				mmap = map[string][]*events.LossSoc{}
			}
			ps, ok := mmap[pID]
			if !ok {
				ps = []*events.LossSoc{}
			}
			ps = append(ps, ls)
			mmap[pID] = ps
			lsEvt[mID] = mmap
		}
	}
	// get by party and market
	pEvts := map[string]map[string][]*eventspb.FundingPayment{}
	for _, pe := range paymentEvts {
		mID := pe.MarketID()
		mmap, ok := pEvts[mID]
		if !ok {
			mmap = map[string][]*eventspb.FundingPayment{}
		}
		for _, fp := range pe.FundingPayments().Payments {
			fps, ok := mmap[fp.PartyId]
			if !ok {
				fps = []*eventspb.FundingPayment{}
			}
			fps = append(fps, fp)
			mmap[fp.PartyId] = fps
		}
		pEvts[mID] = mmap
	}
	// now start matching
	for _, row := range matchers {
		mID, pID := row.Market(), row.Party()
		mmap, ok := pEvts[mID]
		if !ok {
			return fmt.Errorf("could not find funding payment events for market %s", mID)
		}
		ppayments, ok := mmap[pID]
		if !ok {
			return fmt.Errorf("could not find funding payment events for party %s in market %s", pID, mID)
		}
		matched := false
		amt := row.Amount().String()
		for _, fp := range ppayments {
			if fp.Amount == amt {
				matched = true
				break
			}
		}
		if !matched {
			return fmt.Errorf("could not find funding payment of amount %s for party %s in market %s", amt, pID, mID)
		}
		if !checkLoss || !row.CheckLoss() {
			continue
		}
		mloss, ok := lsEvt[mID]
		if !ok {
			return fmt.Errorf("could not find loss socialisation events for market %s", mID)
		}
		pLoss, ok := mloss[pID]
		if !ok {
			return fmt.Errorf("could not find loss socialisation event for party %s in market %s", pID, mID)
		}
		matched = false
		for _, le := range pLoss {
			if !row.matchLossType(le.LossType()) {
				continue
			}
			if !row.matchLossAmount(le.Amount()) {
				continue
			}
			matched = true
			break
		}
		if !matched {
			return fmt.Errorf("could not find loss amount/type %s/%s for party %s in market %s", row.LossAmount().String(), row.LossType().String(), pID, mID)
		}
	}
	return nil
}

func DebugFundingPaymentsEvents(broker *stubs.BrokerStub, log *logging.Logger) {
	paymentEvts := broker.GetFundginPaymentEvents()
	lossSoc := broker.GetLossSoc()
	pEvts := map[string]map[string][]*eventspb.FundingPayment{}
	lsEvt := map[string]map[string][]*events.LossSoc{}
	for _, pe := range paymentEvts {
		mID := pe.MarketID()
		mmap, ok := pEvts[mID]
		if !ok {
			mmap = map[string][]*eventspb.FundingPayment{}
		}
		for _, fp := range pe.FundingPayments().Payments {
			fps, ok := mmap[fp.PartyId]
			if !ok {
				fps = []*eventspb.FundingPayment{}
			}
			fps = append(fps, fp)
			mmap[fp.PartyId] = fps
		}
		pEvts[mID] = mmap
	}
	for _, le := range lossSoc {
		mID, pID := le.MarketID(), le.PartyID()
		// ignore loss socialisation unless they are related to funding payments:
		if mmap, ok := pEvts[mID]; !ok {
			continue
		} else if _, ok := mmap[pID]; !ok {
			// also skip the parties that don't have funding payment events.
			continue
		}
		mmap, ok := lsEvt[mID]
		if !ok {
			mmap = map[string][]*events.LossSoc{}
		}
		// ignore irrelevant parties?
		ps, ok := mmap[pID]
		if !ok {
			ps = []*events.LossSoc{}
		}
		ps = append(ps, le)
		mmap[pID] = ps
		lsEvt[mID] = mmap
	}
	log.Info("DUMPING FUNDING PAYMENTS EVENTS")
	for mID, fpMap := range pEvts {
		log.Infof("Market ID: %s\n", mID)
		for pID, fpe := range fpMap {
			log.Infof("PartyID: %s\n", pID)
			var lSoc []*events.LossSoc
			lossM, ok := lsEvt[mID]
			if ok {
				lSoc = lossM[pID]
			}
			for i, fe := range fpe {
				log.Infof("%d: Amount %s\n", i+1, fe.Amount)
			}
			if len(lSoc) > 0 {
				log.Info("\nLOSS SOCIALISATION:\n")
			}
			for i, le := range lSoc {
				log.Infof("%d: Amount: %s - Type: %s\n", i+1, le.Amount().String(), le.LossType().String())
			}
		}
	}
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

type FundingPaymentsWrapper struct {
	r RowWrapper
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

func parseFundingPaymentsTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"market",
		"amount",
	}, []string{
		"loss type",
		"loss amount",
	})
}

func (f FundingPaymentsWrapper) Party() string {
	return f.r.MustStr("party")
}

func (f FundingPaymentsWrapper) Market() string {
	return f.r.MustStr("market")
}

func (f FundingPaymentsWrapper) Amount() *num.Int {
	return f.r.MustInt("amount")
}

func (f FundingPaymentsWrapper) LossAmount() *num.Int {
	if !f.r.HasColumn("loss amount") {
		return num.IntZero()
	}
	return f.r.MustInt("loss amount")
}

func (f FundingPaymentsWrapper) LossType() types.LossType {
	if !f.r.HasColumn("loss type") {
		return types.LossTypeUnspecified
	}
	return f.r.MustLossType("loss type")
}

func (f FundingPaymentsWrapper) CheckLoss() bool {
	return f.r.HasColumn("loss type") || f.r.HasColumn("loss amount")
}

func (f FundingPaymentsWrapper) matchLossType(t types.LossType) bool {
	if !f.r.HasColumn("loss type") {
		return true
	}
	return f.LossType() == t
}

func (f FundingPaymentsWrapper) matchLossAmount(amt *num.Int) bool {
	if !f.r.HasColumn("loss amount") {
		return true
	}
	return f.LossAmount().EQ(amt)
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
