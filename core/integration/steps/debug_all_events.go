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
