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
		ProposeMarketEnabledFrom: time.Unix(0, vn.ProposeMarketEnabledFrom),
		ProposeAssetEnabledFrom:  time.Unix(0, vn.ProposeAssetEnabledFrom),
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
