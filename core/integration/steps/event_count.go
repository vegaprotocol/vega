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
)

func ExpectingEventsOverStep(broker *stubs.BrokerStub, eventsBeforeStep, expected int) error {
	actual := len(broker.GetAllEvents()) - eventsBeforeStep
	if expected == actual {
		return nil
	}
	return fmt.Errorf("expecting '%d' events generated over the last step, found '%d'", expected, actual)
}

func ExpectingEventsInTheSecenarioSoFar(broker *stubs.BrokerStub, expected int) error {
	events := broker.GetAllEvents()
	actual := len(events)
	if expected == actual {
		return nil
	}
	return fmt.Errorf("expecting '%d' events generated in the scenario so far, found '%d'", expected, actual)
}
