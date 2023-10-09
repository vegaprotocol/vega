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
	"bytes"
	"context"

	"code.vegaprotocol.io/vega/core/types"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/logging"
	checkpoint "code.vegaprotocol.io/vega/protos/vega/checkpoint/v1"

	"code.vegaprotocol.io/vega/libs/proto"
)

func (*Service) Name() types.CheckpointName {
	return types.AssetsCheckpoint
}

func (s *Service) Checkpoint() ([]byte, error) {
	t := &checkpoint.Assets{
		Assets:               s.getEnabled(),
		PendingListingAssets: s.getPendingListed(),
	}
	return proto.Marshal(t)
}

func (s *Service) Load(ctx context.Context, cp []byte) error {
	data := &checkpoint.Assets{}
	if err := proto.Unmarshal(cp, data); err != nil {
		return err
	}
	s.amu.Lock()
	s.pamu.Lock()
	s.pendingAssets = map[string]*Asset{}
	s.pamu.Unlock()
	s.amu.Unlock()
	for _, a := range data.Assets {
		details, _ := types.AssetDetailsFromProto(a.AssetDetails)
		// first check if the asset did get loaded from genesis
		// if yes and the details + IDs are same, then all good
		if existing, ok := s.assets[a.Id]; ok {
			// we know this ID, are the details the same
			// if not, then  there's an error, and we should not overwrite an existing
			// asset, only new ones can be added
			if !bytes.Equal(vgcrypto.Hash([]byte(details.String())), vgcrypto.Hash([]byte(existing.Type().Details.String()))) {
				s.log.Panic("invalid asset loaded from genesis",
					logging.String("id", a.Id),
					logging.String("details-genesis", existing.String()),
					logging.String("details-checkpoint", details.String()))
			}
			continue
		}

		// asset didn't match anything, we need to go through the process to add it.
		s.restoreAsset(ctx, a.Id, details)
		if err := s.Enable(ctx, a.Id); err != nil {
			return err
		}
	}

	// now do pending assets
	for _, pa := range data.PendingListingAssets {
		details, _ := types.AssetDetailsFromProto(pa.AssetDetails)

		// restore it as valid
		s.restoreAsset(ctx, pa.Id, details)

		// set as pending and generate the signatures to list on the contract
		s.SetPendingListing(ctx, pa.Id)
		s.EnactPendingAsset(pa.Id)
	}

	return nil
}

func (s *Service) restoreAsset(ctx context.Context, id string, details *types.AssetDetails) error {
	id, err := s.NewAsset(ctx, id, details)
	if err != nil {
		return err
	}
	pa, _ := s.Get(id)
	if s.isValidator {
		if err := s.validateAsset(pa); err != nil {
			return err
		}
	}
	// always valid now
	pa.SetValid()
	return nil
}

func (s *Service) getEnabled() []*checkpoint.AssetEntry {
	aa := s.GetEnabledAssets()
	if len(aa) == 0 {
		return nil
	}
	ret := make([]*checkpoint.AssetEntry, 0, len(aa))
	for _, a := range aa {
		ret = append(ret, &checkpoint.AssetEntry{
			Id:           a.ID,
			AssetDetails: a.Details.IntoProto(),
		})
	}

	return ret
}

func (s *Service) getPendingListed() []*checkpoint.AssetEntry {
	pd := s.getPendingAssets()
	if len(pd) == 0 {
		return nil
	}
	ret := make([]*checkpoint.AssetEntry, 0, len(pd))
	for _, a := range pd {
		if a.Status != types.AssetStatusPendingListing {
			// we only want enacted but not listed assets
			continue
		}
		ret = append(ret, &checkpoint.AssetEntry{
			Id:           a.ID,
			AssetDetails: a.Details.IntoProto(),
		})
	}
	return ret
}
