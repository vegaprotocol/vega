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
	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/logging"
)

func DebugAllEvents(broker *stubs.BrokerStub, log *logging.Logger) {
	log.Info("DUMPING EVENTS")
	data := broker.GetAllEventsSinceCleared()
	for _, a := range data {
		log.Info(a.Type().String())
	}
}

func DebugLastNEvents(n int, broker *stubs.BrokerStub, log *logging.Logger) {
	log.Infof("DUMPING LAST %d EVENTS", n)
	data := broker.GetAllEvents()
	for i := len(data) - n; i < len(data); i++ {
		log.Info(data[i].Type().String())
	}
}
