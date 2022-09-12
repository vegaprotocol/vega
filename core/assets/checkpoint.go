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
