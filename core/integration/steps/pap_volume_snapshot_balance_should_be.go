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

func PAPVolumeSnapshotShouldBe(
	broker *stubs.BrokerStub,
	marketID, rawBalance string,
) error {
	evt := broker.GetPAPVolumeSnapshotByID(marketID)
	if evt == nil {
		return fmt.Errorf("could not find pap volume snapshot for market")
	}
	if rawBalance != evt.AutomatedPurchaseAnnouncedEvent().Amount {
		return fmt.Errorf("invalid snapshot volume for pap for market id (%s), expected(%s) got(%s)",
			marketID, rawBalance, evt.AutomatedPurchaseAnnouncedEvent().Amount,
		)
	}
	return nil
}
