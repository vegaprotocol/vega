// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
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

func parseEmittedEventsTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"type",
	}, []string{})
}

func errEventNotEmitted(t events.Type) error {
	return fmt.Errorf("event of type %s has not been emitted", t)
}
