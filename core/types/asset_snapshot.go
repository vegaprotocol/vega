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

package types

import (
	"code.vegaprotocol.io/vega/protos/vega"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

type PayloadActiveAssets struct {
	ActiveAssets *ActiveAssets
}

func (p PayloadActiveAssets) IntoProto() *snapshot.Payload_ActiveAssets {
	return &snapshot.Payload_ActiveAssets{
		ActiveAssets: p.ActiveAssets.IntoProto(),
	}
}

func (*PayloadActiveAssets) Namespace() SnapshotNamespace {
	return AssetsSnapshot
}

func (*PayloadActiveAssets) Key() string {
	return "active"
}

func (*PayloadActiveAssets) isPayload() {}

func (p *PayloadActiveAssets) plToProto() interface{} {
	return p.IntoProto()
}

func PayloadActiveAssetsFromProto(paa *snapshot.Payload_ActiveAssets) *PayloadActiveAssets {
	return &PayloadActiveAssets{
		ActiveAssets: ActiveAssetsFromProto(paa.ActiveAssets),
	}
}

type ActiveAssets struct {
	Assets []*Asset
}

func (a ActiveAssets) IntoProto() *snapshot.ActiveAssets {
	ret := &snapshot.ActiveAssets{
		Assets: make([]*vega.Asset, 0, len(a.Assets)),
	}
	for _, a := range a.Assets {
		ret.Assets = append(ret.Assets, a.IntoProto())
	}
	return ret
}

func ActiveAssetsFromProto(aa *snapshot.ActiveAssets) *ActiveAssets {
	ret := ActiveAssets{
		Assets: make([]*Asset, 0, len(aa.Assets)),
	}
	for _, a := range aa.Assets {
		aa, err := AssetFromProto(a)
		if err != nil {
			panic(err)
		}
		ret.Assets = append(ret.Assets, aa)
	}
	return &ret
}

type PayloadPendingAssets struct {
	PendingAssets *PendingAssets
}

func PayloadPendingAssetsFromProto(ppa *snapshot.Payload_PendingAssets) *PayloadPendingAssets {
	return &PayloadPendingAssets{
		PendingAssets: PendingAssetsFromProto(ppa.PendingAssets),
	}
}

func (p PayloadPendingAssets) IntoProto() *snapshot.Payload_PendingAssets {
	return &snapshot.Payload_PendingAssets{
		PendingAssets: p.PendingAssets.IntoProto(),
	}
}

func (*PayloadPendingAssets) Key() string {
	return "pending"
}

func (*PayloadPendingAssets) Namespace() SnapshotNamespace {
	return AssetsSnapshot
}

func (*PayloadPendingAssets) isPayload() {}

func (p *PayloadPendingAssets) plToProto() interface{} {
	return p.IntoProto()
}

type PendingAssets struct {
	Assets []*Asset
}

func PendingAssetsFromProto(aa *snapshot.PendingAssets) *PendingAssets {
	ret := PendingAssets{
		Assets: make([]*Asset, 0, len(aa.Assets)),
	}
	for _, a := range aa.Assets {
		pa, err := AssetFromProto(a)
		if err != nil {
			panic(err)
		}
		ret.Assets = append(ret.Assets, pa)
	}
	return &ret
}

func (a PendingAssets) IntoProto() *snapshot.PendingAssets {
	ret := &snapshot.PendingAssets{
		Assets: make([]*vega.Asset, 0, len(a.Assets)),
	}
	for _, a := range a.Assets {
		ret.Assets = append(ret.Assets, a.IntoProto())
	}
	return ret
}

type PayloadPendingAssetUpdates struct {
	PendingAssetUpdates *PendingAssetUpdates
}

func (p PayloadPendingAssetUpdates) IntoProto() *snapshot.Payload_PendingAssetUpdates {
	return &snapshot.Payload_PendingAssetUpdates{
		PendingAssetUpdates: p.PendingAssetUpdates.IntoProto(),
	}
}

func (*PayloadPendingAssetUpdates) Key() string {
	return "pending_updates"
}

func (*PayloadPendingAssetUpdates) Namespace() SnapshotNamespace {
	return AssetsSnapshot
}

func (*PayloadPendingAssetUpdates) isPayload() {}

func (p *PayloadPendingAssetUpdates) plToProto() interface{} {
	return p.IntoProto()
}

func PayloadPendingAssetUpdatesFromProto(ppa *snapshot.Payload_PendingAssetUpdates) *PayloadPendingAssetUpdates {
	return &PayloadPendingAssetUpdates{
		PendingAssetUpdates: PendingAssetUpdatesFromProto(ppa.PendingAssetUpdates),
	}
}

type PendingAssetUpdates struct {
	Assets []*Asset
}

func (a PendingAssetUpdates) IntoProto() *snapshot.PendingAssetUpdates {
	ret := &snapshot.PendingAssetUpdates{
		Assets: make([]*vega.Asset, 0, len(a.Assets)),
	}
	for _, a := range a.Assets {
		ret.Assets = append(ret.Assets, a.IntoProto())
	}
	return ret
}

func PendingAssetUpdatesFromProto(aa *snapshot.PendingAssetUpdates) *PendingAssetUpdates {
	ret := PendingAssetUpdates{
		Assets: make([]*Asset, 0, len(aa.Assets)),
	}
	for _, a := range aa.Assets {
		pa, err := AssetFromProto(a)
		if err != nil {
			panic(err)
		}
		ret.Assets = append(ret.Assets, pa)
	}
	return &ret
}
