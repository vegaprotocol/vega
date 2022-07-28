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
	buys := fmt.Sprintf("\n\t\t|%25s |%10s |%25s |%10s |", "Order ID", "Offset", "Reference", "Proportion")
	s := fmt.Sprintf("\n\t|%10s |%10s |%20s |%10s |%10s |%20s |", "ID", "Party", "Commitment Amount", "Market", "Fee", "Status")
	for _, lp := range data {
		p := lp.Proto()
		s += fmt.Sprintf("\n\t|%10s |%10s |%20s |%10s |%10s |%20s |", p.Id, p.PartyId, p.CommitmentAmount, p.MarketId, p.Fee, p.Status.String())
		s += buys + " (buy)"
		for _, o := range p.Buys {
			s += fmt.Sprintf("\n\t\t|%25s |%10s |%25s |%10d |", o.OrderId, o.LiquidityOrder.Offset, o.LiquidityOrder.Reference.String(), o.LiquidityOrder.Proportion)
		}
		s += buys + " (sell)"
		for _, o := range p.Sells {
			s += fmt.Sprintf("\n\t\t|%25s |%10s |%25s |%10d |", o.OrderId, o.LiquidityOrder.Offset, o.LiquidityOrder.Reference.String(), o.LiquidityOrder.Proportion)
		}
	}
	log.Infof("%s\n", s)
}
