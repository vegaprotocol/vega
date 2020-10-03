package governance

import types "code.vegaprotocol.io/vega/proto"

// ToEnact wraps the proposal in a type that has a convenient interface
// to quickly work out what change we're dealing with, and get the data
type ToEnact struct {
	p  *types.Proposal
	m  *types.Market
	a  *types.Asset
	n  *types.NetworkParameter
	as *types.AssetSource
	u  *types.UpdateMarket
}

func (t ToEnact) IsNewMarket() bool {
	return (t.m != nil)
}

func (t ToEnact) IsNewAsset() bool {
	a := t.p.Terms.GetNewAsset()
	return (a != nil)
}

func (t ToEnact) IsUpdateMarket() bool {
	return (t.u != nil)
}

func (t ToEnact) IsUpdateNetwork() bool {
	return (t.n != nil)
}

func (t ToEnact) IsNewAssetSource() bool {
	return t.IsNewAsset()
}

func (t *ToEnact) NewMarket() *types.Market {
	return t.m
}

func (t *ToEnact) NewAsset() *types.Asset {
	return t.a
}

func (t *ToEnact) NewAssetSource() *types.AssetSource {
	return t.as
}

func (t *ToEnact) UpdateNetwork() *types.NetworkParameter {
	return t.n
}

func (t *ToEnact) UpdateMarket() *types.UpdateMarket {
	return t.u
}

func (t *ToEnact) Proposal() *types.Proposal {
	return t.p
}
