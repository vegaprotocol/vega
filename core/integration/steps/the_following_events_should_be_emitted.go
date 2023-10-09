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
	"github.com/cucumber/godog"
)

func TheFollowingEventsShouldBeEmitted(broker *stubs.BrokerStub, table *godog.Table) error {
	for _, row := range parseEmittedEventsTable(table) {
		eventType := row.MustEventType("type")

		if len(broker.GetBatch(eventType)) == 0 {
			return errEventNotEmitted(eventType)
		}
	}

	return nil
}

func TheFollowingEventsShouldNotBeEmitted(broker *stubs.BrokerStub, table *godog.Table) error {
	for _, row := range parseEmittedEventsTable(table) {
		eventType := row.MustEventType("type")

		if len(broker.GetBatch(eventType)) > 0 {
			return errEventEmitted(eventType)
		}
	}

	return nil
}

func parseEmittedEventsTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"type",
	}, []string{})
}

func errEventNotEmitted(t events.Type) error {
	return fmt.Errorf("event of type %s has not been emitted", t)
}

func errEventEmitted(t events.Type) error {
	return fmt.Errorf("event of type %s has been emitted", t)
}
