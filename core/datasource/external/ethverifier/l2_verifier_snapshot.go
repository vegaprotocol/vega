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
	"sort"

	"code.vegaprotocol.io/vega/core/datasource/external/ethcall"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/proto"
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
	return types.EthereumOracleVerifierSnapshot
}

func (s *L2Verifiers) Keys() []string {
	return hashKeys
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
		ethOracles.L2EthOracles.ChainIdEthOracles = append(
			ethOracles.L2EthOracles.ChainIdEthOracles,
			&snapshotpb.ChainIdEthOracles{
				SourceChainId: k,
				LastBlock:     v.lastEthBlockPayloadData().IntoProto().EthOracleVerifierLastBlock,
				CallResults:   v.pendingContractCallEventsPayloadData().IntoProto().EthContractCallResults,
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
		return nil, s.restoreState(ctx, pl.L2EthOracles)
	default:
		return nil, types.ErrUnknownSnapshotType
	}
}

func (s *L2Verifiers) restoreState(ctx context.Context, l2EthOracles *snapshotpb.L2EthOracles) error {
	for _, v := range l2EthOracles.ChainIdEthOracles {
		verifier := s.instanciate(v.SourceChainId)

		// might be nil so need proper check first here
		var lastBlock *types.EthBlock
		if v.LastBlock != nil {
			lastBlock.Height = v.LastBlock.BlockHeight
			lastBlock.Time = v.LastBlock.BlockTime
		}
		verifier.restoreLastEthBlock(lastBlock)

		pending := []*ethcall.ContractCallEvent{}

		for _, pr := range v.CallResults.PendingContractCallResult {
			pending = append(pending, &ethcall.ContractCallEvent{
				BlockHeight: pr.BlockHeight,
				BlockTime:   pr.BlockTime,
				SpecId:      pr.SpecId,
				Result:      pr.Result,
				Error:       pr.Error,
			})
		}
		verifier.restorePendingCallEvents(ctx, pending)
	}

	return nil
}

func (s *L2Verifiers) OnStateLoaded(ctx context.Context) error {
	ids := maps.Keys(s.verifiers)
	sort.Strings(ids)

	// restart ethCall engines
	for _, v := range ids {
		s.verifiers[v].OnStateLoaded(ctx)
	}

	return nil
}
