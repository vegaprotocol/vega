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

package vault

import (
	"context"
	"fmt"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
)

func (vs *VaultService) Namespace() types.SnapshotNamespace {
	return types.VaultSnapshot
}

func (vs *VaultService) Keys() []string {
	return []string{(&types.PayloadVault{}).Key()}
}

func (vs *VaultService) GetState(k string) ([]byte, []types.StateProvider, error) {
	vaults := make([]*snapshotpb.VaultState, 0, len(vs.vaultIdToVault))
	for _, vault := range vs.vaultIdToVault {
		shareHolders := make([]*snapshotpb.ShareHolder, 0, len(vault.shareHolders))
		for party, share := range vault.shareHolders {
			shareHolders = append(shareHolders, &snapshotpb.ShareHolder{
				Party: party,
				Share: share.String(),
			})
		}
		sort.Slice(shareHolders, func(i, j int) bool {
			return shareHolders[i].Party < shareHolders[j].Party
		})

		redemptionQueue := make([]*snapshotpb.RedeemRequest, 0, len(vault.redeemQueue))
		for _, rr := range vault.redeemQueue {
			redemptionQueue = append(redemptionQueue, &snapshotpb.RedeemRequest{
				Party:     rr.Party,
				Date:      rr.Date.UnixNano(),
				Amount:    rr.Amount.String(),
				Remaining: rr.Remaining.String(),
				Status:    rr.Status,
			})
		}

		lateRedemptions := make([]*snapshotpb.RedeemRequest, 0, len(vault.lateRedemptions))
		for _, rr := range vault.lateRedemptions {
			lateRedemptions = append(lateRedemptions, &snapshotpb.RedeemRequest{
				Party:     rr.Party,
				Date:      rr.Date.UnixNano(),
				Amount:    rr.Amount.String(),
				Remaining: rr.Remaining.String(),
				Status:    rr.Status,
			})
		}
		vaults = append(vaults, &snapshotpb.VaultState{
			Vault:                   vault.vault.IntoProto(),
			HighWatermark:           vault.highWaterMark.String(),
			InvestedAmount:          vault.investedAmount.String(),
			NextFeeCalc:             vault.nextFeeCalc.UnixNano(),
			Status:                  vault.status,
			NextRedemptionDateIndex: vault.nextFeeCalc.Unix(),
			ShareHolders:            shareHolders,
			RedeemQueue:             redemptionQueue,
			LateRedemptions:         lateRedemptions,
		})
	}

	sort.Slice(vaults, func(i, j int) bool {
		return vaults[i].Vault.VaultId < vaults[j].Vault.VaultId
	})

	payload := &snapshotpb.Payload{
		Data: &snapshotpb.Payload_Vaults{
			Vaults: &snapshotpb.Vault{
				VaultState: vaults,
			},
		},
	}

	serialised, err := proto.Marshal(payload)
	if err != nil {
		return nil, nil, fmt.Errorf("could not serialize vault payload: %w", err)
	}
	return serialised, nil, err
}

func (vs *VaultService) LoadState(_ context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if vs.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	switch data := p.Data.(type) {
	case *types.PayloadVault:
		for _, v := range data.VaultState {
			vault := types.VaultFromProto(v.Vault)
			shareHolders := make(map[string]num.Decimal, len(v.ShareHolders))
			for _, shareHolder := range v.ShareHolders {
				shareHolders[shareHolder.Party] = num.MustDecimalFromString(shareHolder.Share)
			}

			redeemQueue := make([]*RedeemRequest, 0, len(v.RedeemQueue))
			for _, rr := range v.RedeemQueue {
				redeemQueue = append(redeemQueue, &RedeemRequest{
					Party:     rr.Party,
					Date:      time.Unix(0, rr.Date),
					Amount:    num.MustUintFromString(rr.Amount, 10),
					Remaining: num.MustUintFromString(rr.Remaining, 10),
					Status:    rr.Status,
				})
			}
			lateRedemptions := make([]*RedeemRequest, 0, len(v.LateRedemptions))
			for _, rr := range v.LateRedemptions {
				lateRedemptions = append(lateRedemptions, &RedeemRequest{
					Party:     rr.Party,
					Date:      time.Unix(0, rr.Date),
					Amount:    num.MustUintFromString(rr.Amount, 10),
					Remaining: num.MustUintFromString(rr.Remaining, 10),
					Status:    rr.Status,
				})
			}

			vs.vaultIdToVault[vault.ID] = &VaultState{
				log:                     vs.log,
				vault:                   vault,
				collateral:              vs.collateral,
				broker:                  vs.broker,
				status:                  v.Status,
				highWaterMark:           num.MustDecimalFromString(v.HighWatermark),
				nextFeeCalc:             time.Unix(0, v.NextFeeCalc),
				nextRedemptionDateIndex: int(v.NextRedemptionDateIndex),
				investedAmount:          num.MustUintFromString(v.InvestedAmount, 10),
				shareHolders:            shareHolders,
				redeemQueue:             redeemQueue,
				lateRedemptions:         lateRedemptions,
			}
		}
		return nil, nil
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (vs *VaultService) Stopped() bool {
	return false
}

func (e *VaultService) StopSnapshots() {}
