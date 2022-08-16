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

package assets

import (
	"context"
	"sort"

	"code.vegaprotocol.io/vega/core/types"
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
	changedActive            bool
	changedPending           bool
	changedPendingUpdates    bool
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
	sort.SliceStable(enabled, func(i, j int) bool { return enabled[i].ID < enabled[j].ID })
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
	sort.SliceStable(pending, func(i, j int) bool { return pending[i].ID < pending[j].ID })
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
	sort.SliceStable(pendingUpdates, func(i, j int) bool { return pendingUpdates[i].ID < pendingUpdates[j].ID })
	payload := types.Payload{
		Data: &types.PayloadPendingAssetUpdates{
			PendingAssetUpdates: &types.PendingAssetUpdates{
				Assets: pendingUpdates,
			},
		},
	}

	return proto.Marshal(payload.IntoProto())
}

func (s *Service) serialiseK(k string, serialFunc func() ([]byte, error), dataField *[]byte, changedField *bool) ([]byte, error) {
	if !s.HasChanged(k) {
		if dataField == nil {
			return nil, nil
		}
		return *dataField, nil
	}
	data, err := serialFunc()
	if err != nil {
		return nil, err
	}
	*dataField = data
	*changedField = false
	return data, nil
}

// get the serialised form and hash of the given key.
func (s *Service) serialise(k string) ([]byte, error) {
	switch k {
	case activeKey:
		return s.serialiseK(k, s.serialiseActive, &s.ass.serialisedActive, &s.ass.changedActive)
	case pendingKey:
		return s.serialiseK(k, s.serialisePending, &s.ass.serialisedPending, &s.ass.changedPending)
	case pendingUpdatesKey:
		return s.serialiseK(k, s.serialisePendingUpdates, &s.ass.serialisedPendingUpdates, &s.ass.changedPendingUpdates)
	default:
		return nil, types.ErrSnapshotKeyDoesNotExist
	}
}

func (s *Service) HasChanged(k string) bool {
	// switch k {
	// case activeKey:
	// 	return s.ass.changedActive
	// case pendingKey:
	// 	return s.ass.changedPending
	// case pendingUpdatesKey:
	// 	return s.ass.changedPending
	// default:
	// 	return false
	// }
	return true
}

func (s *Service) GetState(k string) ([]byte, []types.StateProvider, error) {
	state, err := s.serialise(k)
	return state, nil, err
}

func (s *Service) LoadState(ctx context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if s.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}
	// see what we're reloading
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
	s.ass.changedActive = false
	s.ass.serialisedActive, err = proto.Marshal(p.IntoProto())

	return err
}

func (s *Service) restorePending(ctx context.Context, pending *types.PendingAssets, p *types.Payload) error {
	var err error
	s.pendingAssets = map[string]*Asset{}
	for _, p := range pending.Assets {
		assetID, err := s.NewAsset(ctx, p.ID, p.Details)
		if err != nil {
			return err
		}

		if p.Status == types.AssetStatusPendingListing {
			s.SetPendingListing(ctx, assetID)
		}
	}

	s.ass.changedPending = false
	s.ass.serialisedPending, err = proto.Marshal(p.IntoProto())

	return err
}

func (s *Service) restorePendingUpdates(_ context.Context, pending *types.PendingAssetUpdates, p *types.Payload) error {
	var err error
	s.pendingAssetUpdates = map[string]*Asset{}
	for _, p := range pending.Assets {
		if err = s.StageAssetUpdate(p); err != nil {
			return err
		}
	}
	s.ass.changedPendingUpdates = false
	s.ass.serialisedPendingUpdates, err = proto.Marshal(p.IntoProto())

	return err
}
