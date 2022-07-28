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

package entities

import (
	"time"

	"code.vegaprotocol.io/protos/vega"
)

type NetworkLimits struct {
	VegaTime                 time.Time
	CanProposeMarket         bool
	CanProposeAsset          bool
	BootstrapFinished        bool
	ProposeMarketEnabled     bool
	ProposeAssetEnabled      bool
	BootstrapBlockCount      int32
	GenesisLoaded            bool
	ProposeMarketEnabledFrom time.Time
	ProposeAssetEnabledFrom  time.Time
}

func NetworkLimitsFromProto(vn *vega.NetworkLimits) NetworkLimits {
	return NetworkLimits{
		CanProposeMarket:         vn.CanProposeMarket,
		CanProposeAsset:          vn.CanProposeAsset,
		BootstrapFinished:        vn.BootstrapFinished,
		ProposeMarketEnabled:     vn.ProposeMarketEnabled,
		ProposeAssetEnabled:      vn.ProposeAssetEnabled,
		BootstrapBlockCount:      int32(vn.BootstrapBlockCount),
		GenesisLoaded:            vn.GenesisLoaded,
		ProposeMarketEnabledFrom: NanosToPostgresTimestamp(vn.ProposeMarketEnabledFrom),
		ProposeAssetEnabledFrom:  NanosToPostgresTimestamp(vn.ProposeAssetEnabledFrom),
	}
}

func (nl *NetworkLimits) ToProto() *vega.NetworkLimits {
	return &vega.NetworkLimits{
		CanProposeMarket:         nl.CanProposeMarket,
		CanProposeAsset:          nl.CanProposeAsset,
		BootstrapFinished:        nl.BootstrapFinished,
		ProposeMarketEnabled:     nl.ProposeMarketEnabled,
		ProposeAssetEnabled:      nl.ProposeAssetEnabled,
		BootstrapBlockCount:      uint32(nl.BootstrapBlockCount),
		GenesisLoaded:            nl.GenesisLoaded,
		ProposeMarketEnabledFrom: nl.ProposeMarketEnabledFrom.UnixNano(),
		ProposeAssetEnabledFrom:  nl.ProposeAssetEnabledFrom.UnixNano(),
	}
}
