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

package ethverifier

import (
	"context"
	"slices"
	"sort"

	"code.vegaprotocol.io/vega/core/datasource/external/ethcall"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"golang.org/x/exp/maps"
)

var (
	l2StateKey = (&types.PayloadL2EthOracles{}).Key()
	l2HashKeys = []string{
		l2StateKey,
	}
)

func (s *L2Verifiers) Namespace() types.SnapshotNamespace {
	return types.L2EthereumOraclesSnapshot
}

func (s *L2Verifiers) Keys() []string {
	return l2HashKeys
}

func (s *L2Verifiers) Stopped() bool {
	return false
}

func (s *L2Verifiers) GetState(k string) ([]byte, []types.StateProvider, error) {
	if k != s.Keys()[0] {
		return nil, nil, types.ErrInvalidSnapshotNamespace
	}

	ethOracles := &types.PayloadL2EthOracles{
		L2EthOracles: &snapshotpb.L2EthOracles{},
	}

	for k, v := range s.verifiers {
		s.log.Debug("serialising state for evm verifier", logging.String("source-chain-id", k))

		slice := make([]*snapshotpb.EthVerifierBucket, 0, v.ackedEvts.Size())
		iter := v.ackedEvts.events.Iterator()
		for iter.Next() {
			v := (iter.Value().(*ackedEvtBucket))
			hashes := maps.Keys(v.hashes)
			slices.Sort(hashes)
			slice = append(slice, &snapshotpb.EthVerifierBucket{
				Ts:     v.ts,
				Hashes: hashes,
			})
		}

		ethOracles.L2EthOracles.ChainIdEthOracles = append(
			ethOracles.L2EthOracles.ChainIdEthOracles,
			&snapshotpb.ChainIdEthOracles{
				SourceChainId: k,
				LastBlock:     v.ethBlockPayloadData(v.lastBlock).IntoProto().EthOracleVerifierLastBlock,
				CallResults:   v.pendingContractCallEventsPayloadData().IntoProto().EthContractCallResults,
				Misc: &snapshotpb.EthOracleVerifierMisc{
					Buckets:    slice,
					PatchBlock: v.ethBlockPayloadData(v.patchBlock).IntoProto().EthOracleVerifierLastBlock,
				},
			},
		)
	}

	sort.Slice(ethOracles.L2EthOracles.ChainIdEthOracles, func(i, j int) bool {
		return ethOracles.L2EthOracles.ChainIdEthOracles[i].SourceChainId < ethOracles.L2EthOracles.ChainIdEthOracles[j].SourceChainId
	})

	pl := types.Payload{
		Data: ethOracles,
	}

	data, err := proto.Marshal(pl.IntoProto())

	return data, nil, err
}

func (s *L2Verifiers) LoadState(ctx context.Context, payload *types.Payload) ([]types.StateProvider, error) {
	if s.Namespace() != payload.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	switch pl := payload.Data.(type) {
	case *types.PayloadL2EthOracles:
		s.restoreState(ctx, pl.L2EthOracles)
		return nil, nil
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (s *L2Verifiers) restoreState(ctx context.Context, l2EthOracles *snapshotpb.L2EthOracles) {
	for _, v := range l2EthOracles.ChainIdEthOracles {
		verifier, ok := s.verifiers[v.SourceChainId]
		if !ok {
			s.log.Panic("evm verifier for chain in snapshot, but not instantiated by network-parameter", logging.String("source-chain-id", v.SourceChainId))
		}

		s.log.Info("restoring evm verifier", logging.String("source-chain-id", v.SourceChainId))
		// might be nil so need proper check first here
		var lastBlock *types.EthBlock
		if v.LastBlock != nil {
			lastBlock = &types.EthBlock{
				Height: v.LastBlock.BlockHeight,
				Time:   v.LastBlock.BlockTime,
			}
		}

		// do it once always
		verifier.restoreLastEthBlock(ctx, lastBlock)

		// this is the block of the upgrade
		// we only initialize this the patchBlock and lastBlock
		if v.Misc == nil {
			if lastBlock != nil {
				// no patchBlock, set it to the last Block
				verifier.restorePatchBlock(ctx, &types.EthBlock{
					Height: lastBlock.Height,
					Time:   lastBlock.Time,
				})
			}
		} else if v.Misc != nil {
			// only run this if the misc exists, which might
			// not be the case on a new upgrade after it's
			// introduced
			var patchBlock *types.EthBlock
			if v.Misc.PatchBlock != nil {
				patchBlock = &types.EthBlock{
					Height: v.Misc.PatchBlock.BlockHeight,
					Time:   v.Misc.PatchBlock.BlockTime,
				}
			}

			verifier.restorePatchBlock(ctx, patchBlock)
			verifier.restoreSeen(ctx, v.Misc.Buckets)
		}
		pending := []*ethcall.ContractCallEvent{}

		for _, pr := range v.CallResults.PendingContractCallResult {
			pending = append(pending, &ethcall.ContractCallEvent{
				BlockHeight:   pr.BlockHeight,
				BlockTime:     pr.BlockTime,
				SpecId:        pr.SpecId,
				Result:        pr.Result,
				Error:         pr.Error,
				SourceChainID: pr.ChainId,
			})
		}
		verifier.restorePendingCallEvents(ctx, pending)
	}
}

func (s *L2Verifiers) OnStateLoaded(ctx context.Context) error {
	ids := maps.Keys(s.verifiers)
	sort.Strings(ids)

	// restart ethCall engines
	for _, v := range ids {
		s.log.Info("calling OnStateLoaded for evm verifier", logging.String("source-chain-id", v))
		s.verifiers[v].OnStateLoaded(ctx)
	}

	return nil
}
