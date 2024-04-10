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

	"code.vegaprotocol.io/vega/core/types"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
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

		// at this point asset is always valid because we've loaded from a snapshot and have validated it when it was proposed
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

func (s *Service) OnStateLoaded(ctx context.Context) error {
	if !vgcontext.InProgressUpgrade(ctx) || s.isValidator {
		return nil
	}

	// note that non-validator nodes do not know the chain-id for the bridges until the network parameters are propagated, but also *validator* nodes need to
	// restore assets before the network parameters. So for the non-validator nodes only, we have to do the migration to include the chain-id here, after everything
	// else is restored.
	s.log.Info("migrating chain-id in existing active assets for non-validator nodes")
	for k, a := range s.assets {
		eth, ok := a.ERC20()
		if !ok || eth.ChainID() != "" {
			continue
		}
		s.log.Info("setting chain-id for active asset",
			logging.String("asset-id", k),
			logging.String("chain-id", s.primaryEthChainID),
		)
		eth.SetChainID(s.primaryEthChainID)
	}

	s.log.Info("migrating chain-id in existing pending assets for non-validator nodes")
	for k, p := range s.pendingAssets {
		eth, ok := p.ERC20()
		if !ok || eth.ChainID() != "" {
			continue
		}
		s.log.Info("setting chain-id for pending asset",
			logging.String("asset-id", k),
			logging.String("chain-id", s.primaryEthChainID),
		)
		eth.SetChainID(s.primaryEthChainID)
	}

	s.log.Info("migrating chain-id in existing update-pending assets for non-validator nodes")
	for k, pu := range s.pendingAssetUpdates {
		eth, ok := pu.ERC20()
		if !ok || eth.ChainID() != "" {
			continue
		}
		s.log.Info("setting chain-id for pending update asset",
			logging.String("asset-id", k),
			logging.String("chain-id", s.primaryEthChainID),
		)
		eth.SetChainID(s.primaryEthChainID)
	}

	return nil
}

func (s *Service) applyMigrations(ctx context.Context, p *types.Asset) {
	// TODO when we know what versions we are upgrading from we can put in a upgrade from tag
	if vgcontext.InProgressUpgrade(ctx) && s.isValidator {
		// Prior the introduction of the second bridge, existing assets did not track
		// the chain ID they originated from. So, when loaded, assets without a chain
		// ID are automatically considered to originate from Ethereum Mainnet.
		if erc20 := p.Details.GetERC20(); erc20 != nil && erc20.ChainID == "" {
			s.log.Info("migrating chain-id in existin asset for validator nodes",
				logging.String("asset-id", p.ID),
				logging.String("chain-id", s.primaryEthChainID),
			)
			erc20.ChainID = s.primaryEthChainID
		}
	}
}
