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

package pow

import (
	"context"

	"code.vegaprotocol.io/vega/core/types"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
	"github.com/golang/protobuf/proto"
)

func (e *Engine) Namespace() types.SnapshotNamespace {
	return types.PoWSnapshot
}

func (e *Engine) Keys() []string {
	return e.hashKeys
}

func (e *Engine) Stopped() bool {
	return false
}

// get the serialised form and hash of the given key.
func (e *Engine) serialise() ([]byte, error) {
	payloadProofOfWork := &types.PayloadProofOfWork{
		BlockHeight:   e.blockHeight[:ringSize],
		BlockHash:     e.blockHash[:ringSize],
		HeightToTx:    e.heightToTx,
		HeightToTid:   e.heightToTid,
		BannedParties: e.bannedParties,
		ActiveParams:  e.paramsToSnapshotParams(),
	}
	payload := types.Payload{
		Data: payloadProofOfWork,
	}

	data, err := proto.Marshal(payload.IntoProto())
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (e *Engine) GetState(k string) ([]byte, []types.StateProvider, error) {
	state, err := e.serialise()
	return state, nil, err
}

func (e *Engine) paramsToSnapshotParams() []*snapshot.ProofOfWorkParams {
	params := make([]*snapshot.ProofOfWorkParams, 0, len(e.activeParams))
	for _, p := range e.activeParams {
		until := int64(-1)
		if p.untilBlock != nil {
			until = int64(*p.untilBlock)
		}
		params = append(params, &snapshot.ProofOfWorkParams{
			SpamPowNumberOfPastBlocks:   p.spamPoWNumberOfPastBlocks,
			SpamPowDifficulty:           uint32(p.spamPoWDifficulty),
			SpamPowHashFunction:         p.spamPoWHashFunction,
			SpamPowNumberOfTxPerBlock:   p.spamPoWNumberOfTxPerBlock,
			SpamPowIncreasingDifficulty: p.spamPoWIncreasingDifficulty,
			FromBlock:                   p.fromBlock,
			UntilBlock:                  until,
		})
	}
	return params
}

func (e *Engine) snapshotParamsToParams(activeParams []*snapshot.ProofOfWorkParams) []*params {
	pars := make([]*params, 0, len(activeParams))
	for _, p := range activeParams {
		param := &params{
			spamPoWNumberOfPastBlocks:   p.SpamPowNumberOfPastBlocks,
			spamPoWDifficulty:           uint(p.SpamPowDifficulty),
			spamPoWHashFunction:         p.SpamPowHashFunction,
			spamPoWNumberOfTxPerBlock:   p.SpamPowNumberOfTxPerBlock,
			spamPoWIncreasingDifficulty: p.SpamPowIncreasingDifficulty,
			fromBlock:                   p.FromBlock,
			untilBlock:                  nil,
		}
		if p.UntilBlock >= 0 {
			param.untilBlock = new(uint64)
			*param.untilBlock = uint64(p.UntilBlock)
		}
		pars = append(pars, param)
	}
	return pars
}

func (e *Engine) LoadState(ctx context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if e.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}
	pl := p.Data.(*types.PayloadProofOfWork)
	e.bannedParties = pl.BannedParties
	copy(e.blockHash[:], pl.BlockHash[:ringSize])
	copy(e.blockHeight[:], pl.BlockHeight[:ringSize])
	e.heightToTx = pl.HeightToTx
	e.heightToTid = pl.HeightToTid
	e.seenTx = map[string]struct{}{}
	e.seenTid = map[string]struct{}{}
	for _, block := range e.heightToTid {
		for _, v := range block {
			e.seenTid[v] = struct{}{}
		}
	}
	for _, block := range e.heightToTx {
		for _, v := range block {
			e.seenTx[v] = struct{}{}
		}
	}
	e.activeParams = e.snapshotParamsToParams(pl.ActiveParams)
	e.activeStates = make([]*state, 0, len(e.activeParams))
	for i := 0; i < len(e.activeParams); i++ {
		s := state{}
		s.blockPartyToObservedDifficulty = map[string]uint{}
		s.blockPartyToSeenCount = map[string]uint{}
		e.activeStates = append(e.activeStates, &s)
	}
	return nil, nil
}
