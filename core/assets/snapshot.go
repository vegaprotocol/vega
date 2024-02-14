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

package assets

import (
	"context"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/proto"
)

var (
	activeKey         = (&types.PayloadActiveAssets{}).Key()
	pendingKey        = (&types.PayloadPendingAssets{}).Key()
	pendingUpdatesKey = (&types.PayloadPendingAssetUpdates{}).Key()

	hashKeys = []string{
		activeKey,
		pendingKey,
		pendingUpdatesKey,
	}
)

type assetsSnapshotState struct {
	serialisedActive         []byte
	serialisedPending        []byte
	serialisedPendingUpdates []byte
}

func (s *Service) Namespace() types.SnapshotNamespace {
	return types.AssetsSnapshot
}

func (s *Service) Keys() []string {
	return hashKeys
}

func (s *Service) Stopped() bool {
	return false
}

func (s *Service) serialiseActive() ([]byte, error) {
	enabled := s.GetEnabledAssets()
	payload := types.Payload{
		Data: &types.PayloadActiveAssets{
			ActiveAssets: &types.ActiveAssets{
				Assets: enabled,
			},
		},
	}
	return proto.Marshal(payload.IntoProto())
}

func (s *Service) serialisePending() ([]byte, error) {
	pending := s.getPendingAssets()
	payload := types.Payload{
		Data: &types.PayloadPendingAssets{
			PendingAssets: &types.PendingAssets{
				Assets: pending,
			},
		},
	}

	return proto.Marshal(payload.IntoProto())
}

func (s *Service) serialisePendingUpdates() ([]byte, error) {
	pendingUpdates := s.getPendingAssetUpdates()
	payload := types.Payload{
		Data: &types.PayloadPendingAssetUpdates{
			PendingAssetUpdates: &types.PendingAssetUpdates{
				Assets: pendingUpdates,
			},
		},
	}

	return proto.Marshal(payload.IntoProto())
}

func (s *Service) serialiseK(serialFunc func() ([]byte, error), dataField *[]byte) ([]byte, error) {
	data, err := serialFunc()
	if err != nil {
		return nil, err
	}
	*dataField = data
	return data, nil
}

func (s *Service) serialise(k string) ([]byte, error) {
	switch k {
	case activeKey:
		return s.serialiseK(s.serialiseActive, &s.ass.serialisedActive)
	case pendingKey:
		return s.serialiseK(s.serialisePending, &s.ass.serialisedPending)
	case pendingUpdatesKey:
		return s.serialiseK(s.serialisePendingUpdates, &s.ass.serialisedPendingUpdates)
	default:
		return nil, types.ErrSnapshotKeyDoesNotExist
	}
}

func (s *Service) GetState(k string) ([]byte, []types.StateProvider, error) {
	state, err := s.serialise(k)
	return state, nil, err
}

func (s *Service) LoadState(ctx context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if s.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	switch pl := p.Data.(type) {
	case *types.PayloadActiveAssets:
		return nil, s.restoreActive(ctx, pl.ActiveAssets, p)
	case *types.PayloadPendingAssets:
		return nil, s.restorePending(ctx, pl.PendingAssets, p)
	case *types.PayloadPendingAssetUpdates:
		return nil, s.restorePendingUpdates(ctx, pl.PendingAssetUpdates, p)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (s *Service) restoreActive(ctx context.Context, active *types.ActiveAssets, p *types.Payload) error {
	var err error
	s.assets = map[string]*Asset{}
	for _, p := range active.Assets {
		s.applyMigrations(ctx, p)

		if _, err = s.NewAsset(ctx, p.ID, p.Details); err != nil {
			return err
		}

		pa, _ := s.Get(p.ID)
		if s.isValidator {
			if err = s.validateAsset(pa); err != nil {
				return err
			}
		}
		// at this point asset is always valid
		pa.SetValid()

		if err = s.Enable(ctx, p.ID); err != nil {
			return err
		}
	}
	s.ass.serialisedActive, err = proto.Marshal(p.IntoProto())

	return err
}

func (s *Service) restorePending(ctx context.Context, pending *types.PendingAssets, p *types.Payload) error {
	var err error
	s.pendingAssets = map[string]*Asset{}
	for _, p := range pending.Assets {
		s.applyMigrations(ctx, p)

		assetID, err := s.NewAsset(ctx, p.ID, p.Details)
		if err != nil {
			return err
		}

		if p.Status == types.AssetStatusPendingListing {
			s.SetPendingListing(ctx, assetID)
		}
	}

	s.ass.serialisedPending, err = proto.Marshal(p.IntoProto())

	return err
}

func (s *Service) restorePendingUpdates(ctx context.Context, pending *types.PendingAssetUpdates, p *types.Payload) error {
	var err error
	s.pendingAssetUpdates = map[string]*Asset{}
	for _, p := range pending.Assets {
		s.applyMigrations(ctx, p)

		if err = s.StageAssetUpdate(p); err != nil {
			return err
		}
	}
	s.ass.serialisedPendingUpdates, err = proto.Marshal(p.IntoProto())

	return err
}

func (s *Service) applyMigrations(ctx context.Context, p *types.Asset) {
	if vgcontext.InProgressUpgradeFrom(ctx, "v0.74.0") {
		// Prior the introduction of the second bridge, existing assets did not track
		// the chain ID they originated from. So, when loaded, assets without a chain
		// ID are automatically considered to originate from Ethereum Mainnet.
		if erc20 := p.Details.GetERC20(); erc20 != nil && erc20.ChainID == "" {
			erc20.ChainID = s.primaryEthChainID
			// Ensure the assets are updated in the data-node.
			s.broker.Send(events.NewAssetEvent(ctx, *p))
		}
	}
}
