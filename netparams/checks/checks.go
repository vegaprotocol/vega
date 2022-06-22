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

package checks

import (
	"errors"
	"fmt"

	"code.vegaprotocol.io/data-node/logging"
	types "code.vegaprotocol.io/protos/vega"
)

type Assets interface {
	IsEnabled(asset string) bool
}

func MarginScalingFactor() func(interface{}) error {
	return func(v interface{}) error {
		sf := v.(*types.ScalingFactors)
		if sf.SearchLevel >= sf.InitialMargin || sf.InitialMargin >= sf.CollateralRelease {
			return errors.New("invalid scaling factors (searchLevel < initialMargin < collateralRelease)")
		}
		return nil
	}
}

func GovernanceAssetUpdate(
	log *logging.Logger,
	assets Assets,
) func(value string) error {
	return func(value string) error {
		if !assets.IsEnabled(value) {
			log.Debug("tried to push a governance update with an non-enabled asset",
				logging.String("asset-id", value))
			return fmt.Errorf("invalid asset %v", value)
		}

		return nil
	}
}

func EthereumConfig() func(interface{}) error {
	return func(v interface{}) error {
		ecfg := v.(*types.EthereumConfig)
		if len(ecfg.NetworkId) <= 0 {
			return errors.New("missing ethereum config network id")
		}
		if len(ecfg.ChainId) <= 0 {
			return errors.New("missing ethereum config chain id")
		}
		if len(ecfg.CollateralBridgeContract.Address) <= 0 {
			return errors.New("missing ethereum collateral config bridge address")
		}
		if ecfg.Confirmations == 0 {
			return errors.New("ethereum config confirmation must be > 0")
		}
		return nil
	}
}
