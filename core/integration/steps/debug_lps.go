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

	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/logging"
)

func DebugLPs(broker *stubs.BrokerStub, log *logging.Logger) {
	log.Info("DUMPING LIQUIDITY PROVISION EVENTS")
	data := broker.GetLPEvents()
	for _, lp := range data {
		p := lp.Proto()
		log.Infof("LP %s, %#v\n", p.String(), p)
	}
}

func DebugLPDetail(log *logging.Logger, broker *stubs.BrokerStub) {
	log.Info("DUMPING DETAILED LIQUIDITY PROVISION EVENTS")
	data := broker.GetLPEvents()
	s := fmt.Sprintf("\n\t|%10s |%10s |%20s |%10s |%10s |%20s |", "ID", "Party", "Commitment Amount", "Market", "Fee", "Status")
	for _, lp := range data {
		p := lp.Proto()
		s += fmt.Sprintf("\n\t|%10s |%10s |%20s |%10s |%10s |%20s |", p.Id, p.PartyId, p.CommitmentAmount, p.MarketId, p.Fee, p.Status.String())
	}
	log.Infof("%s\n", s)
}
