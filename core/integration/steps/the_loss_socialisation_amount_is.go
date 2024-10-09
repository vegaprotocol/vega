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
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"

	"github.com/cucumber/godog"
)

func TheLossSocialisationAmountsAre(broker *stubs.BrokerStub, table *godog.Table) error {
	lsMkt := getLossSocPerMarket(broker)
	for _, r := range parseLossSocTable(table) {
		lsr := lossSocRow{r: r}
		mevts, ok := lsMkt[lsr.Market()]
		if !ok {
			return fmt.Errorf("no loss socialisation events found for market %s", lsr.Market())
		}
		parties := map[string]types.LossType{}
		for _, e := range mevts {
			if lsr.Amount().EQ(e.Amount()) {
				parties[e.PartyID()] = e.LossType()
			}
		}
		if c := lsr.Count(); c != -1 {
			if len(parties) != c {
				return fmt.Errorf("expected %d loss socialisation events for market %s and amount %s, instead found %d", c, lsr.Market(), lsr.Amount().String(), len(parties))
			}
		}
		for _, p := range lsr.Party() {
			lt, ok := parties[p]
			if !ok {
				return fmt.Errorf("no loss socialisation found for party %s on market %s for amount %s (type: %s)", p, lsr.Market(), lsr.Amount().String(), lsr.Type().String())
			}
			if !lsr.matchesType(lt) {
				return fmt.Errorf("loss socialisation for party %s on market %s for amount %s is of type %s, not %s", p, lsr.Market(), lsr.Amount().String(), lt.String(), lsr.Type().String())
			}
		}
	}
	return nil
}

func DebugLossSocialisationEvents(broker *stubs.BrokerStub, log *logging.Logger) error {
	lsEvts := getLossSocPerMarket(broker)
	for mkt, evts := range lsEvts {
		log.Infof("\nLoss socialisation events for market %s:", mkt)
		for _, e := range evts {
			log.Infof(
				"Party: %s - Amount: %s",
				e.PartyID(),
				e.Amount().String(),
			)
		}
		log.Info("----------------------------------------------------------------------------")
	}
	return nil
}

func getLossSocPerMarket(broker *stubs.BrokerStub) map[string][]*events.LossSoc {
	evts := broker.GetLossSoc()
	ret := map[string][]*events.LossSoc{}
	for _, e := range evts {
		mkt := e.MarketID()
		mevts, ok := ret[mkt]
		if !ok {
			mevts = []*events.LossSoc{}
		}
		ret[mkt] = append(mevts, e)
	}
	return ret
}

func parseLossSocTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"market",
		"amount",
	}, []string{
		"party",
		"count",
		"type",
	})
}

type lossSocRow struct {
	r RowWrapper
}

func (l lossSocRow) Market() string {
	return l.r.MustStr("market")
}

func (l lossSocRow) Amount() *num.Int {
	return l.r.MustInt("amount")
}

func (l lossSocRow) Party() []string {
	if l.r.HasColumn("party") {
		return l.r.MustStrSlice("party", ",")
	}
	return nil
}

func (l lossSocRow) Count() int {
	if l.r.HasColumn("count") {
		return int(l.r.MustI64("count"))
	}
	return -1
}

func (l lossSocRow) matchesType(t types.LossType) bool {
	if l.r.HasColumn("type") {
		return l.Type() == t
	}
	return true
}

func (l lossSocRow) Type() types.LossType {
	if !l.r.HasColumn("type") {
		return types.LossTypeUnspecified
	}
	return l.r.MustLossType("type")
}
