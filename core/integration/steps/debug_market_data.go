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
