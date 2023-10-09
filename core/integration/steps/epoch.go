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

func TheCurrentEpochIs(broker *stubs.BrokerStub, epoch string) error {
	seq, err := U64(epoch)
	if err != nil {
		return err
	}
	last := broker.GetCurrentEpoch()

	// If we haven't had an epoch event yet
	// assume we are on epoch 0
	ce := uint64(0)
	if last != nil {
		ce = last.Epoch().GetSeq()
	}
	if ce != seq {
		return fmt.Errorf("expected current epoch to be %d, instead saw %d", seq, ce)
	}
	return nil
}
