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
	"errors"

	"code.vegaprotocol.io/vega/logging"
)

func DebugMarketData(
	exec Execution,
	log *logging.Logger,
	market string,
) error {
	log.Info("DUMPING MARKET DATA")
	marketData, err := exec.GetMarketData(market)
	if err != nil {
		return errors.New("market not found")
	}
	log.Info(marketData.String())

	return nil
}
