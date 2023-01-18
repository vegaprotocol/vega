// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
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

	"code.vegaprotocol.io/vega/protos/vega"
)

type NetworkLimits struct {
	TxHash                   TxHash
	VegaTime                 time.Time
	CanProposeMarket         bool
	CanProposeAsset          bool
	ProposeMarketEnabled     bool
	ProposeAssetEnabled      bool
	GenesisLoaded            bool
	ProposeMarketEnabledFrom time.Time
	ProposeAssetEnabledFrom  time.Time
}

func NetworkLimitsFromProto(vn *vega.NetworkLimits, txHash TxHash) NetworkLimits {
	return NetworkLimits{
		TxHash:                   txHash,
		CanProposeMarket:         vn.CanProposeMarket,
		CanProposeAsset:          vn.CanProposeAsset,
		ProposeMarketEnabled:     vn.ProposeMarketEnabled,
		ProposeAssetEnabled:      vn.ProposeAssetEnabled,
		GenesisLoaded:            vn.GenesisLoaded,
		ProposeMarketEnabledFrom: NanosToPostgresTimestamp(vn.ProposeMarketEnabledFrom),
		ProposeAssetEnabledFrom:  NanosToPostgresTimestamp(vn.ProposeAssetEnabledFrom),
	}
}

func (nl *NetworkLimits) ToProto() *vega.NetworkLimits {
	return &vega.NetworkLimits{
		CanProposeMarket:         nl.CanProposeMarket,
		CanProposeAsset:          nl.CanProposeAsset,
		ProposeMarketEnabled:     nl.ProposeMarketEnabled,
		ProposeAssetEnabled:      nl.ProposeAssetEnabled,
		GenesisLoaded:            nl.GenesisLoaded,
		ProposeMarketEnabledFrom: nl.ProposeMarketEnabledFrom.UnixNano(),
		ProposeAssetEnabledFrom:  nl.ProposeAssetEnabledFrom.UnixNano(),
	}
}
