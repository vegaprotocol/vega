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
