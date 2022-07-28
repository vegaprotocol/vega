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

func DebugAuctionEvents(broker *stubs.BrokerStub, log *logging.Logger) {
	log.Info("DUMPING AUCTION EVENTS")
	data := broker.GetAuctionEvents()
	for _, a := range data {
		log.Info(fmt.Sprintf("AuctionEvent summary: %s, %#v\n", a.MarketEvent(), a))
	}
}
