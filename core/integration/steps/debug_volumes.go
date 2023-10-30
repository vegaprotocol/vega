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

func DebugVolumesForMarket(log *logging.Logger, broker *stubs.BrokerStub, marketID string) error {
	sell, buy := broker.GetBookDepth(marketID)
	log.Info("SELL volume:")
	for price, vol := range sell {
		log.Info(fmt.Sprintf("Price %s: %d\n", price, vol))
	}
	log.Info("BUY volume:")
	for price, vol := range buy {
		log.Info(fmt.Sprintf("Price %s: %d\n", price, vol))
	}
	return nil
}

func DebugVolumesForMarketDetail(log *logging.Logger, broker *stubs.BrokerStub, marketID string) error {
	sell, buy := broker.GetActiveOrderDepth(marketID)
	s := fmt.Sprintf("\nSELL orders:\n\t|%20s |%10s |%10s |%40s |", "Party", "Volume", "Remaining", "Price")
	for _, o := range sell {
		s += fmt.Sprintf("\n\t|%20s |%10d |%10d |%40s |", o.PartyId, o.Size, o.Remaining, o.Price)
	}
	s += fmt.Sprintf("\nBUY orders:\n\t|%20s |%10s |%10s |%40s |", "Party", "Volume", "Remaining", "Price")
	for _, o := range buy {
		s += fmt.Sprintf("\n\t|%20s |%10d |%10d |%40s |", o.PartyId, o.Size, o.Remaining, o.Price)
	}
	log.Infof("%s\n", s)
	return nil
}
