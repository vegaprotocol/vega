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

package pow

import (
	"context"
	"sort"

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
	nonceHeights := map[uint64][]*snapshot.NonceRef{}

	for k, v := range e.heightToNonceRef {
		refs := make([]*snapshot.NonceRef, 0, len(v))
		for _, ref := range v {
			refs = append(refs, &snapshot.NonceRef{Party: ref.party, Nonce: ref.nonce})
		}
		nonceHeights[k] = refs
	}

	payloadProofOfWork := &types.PayloadProofOfWork{
		BlockHeight:      e.blockHeight[:ringSize],
		BlockHash:        e.blockHash[:ringSize],
		HeightToTx:       e.heightToTx,
		HeightToTid:      e.heightToTid,
		HeightToNonceRef: nonceHeights,
		ActiveParams:     e.paramsToSnapshotParams(),
		ActiveStates:     e.statesToSnapshotStates(),
		LastPruningBlock: e.lastPruningBlock,
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

func (e *Engine) blocksState(state map[uint64]map[string]*partyStateForBlock) []*snapshot.ProofOfWorkBlockState {
	states := make([]*snapshot.ProofOfWorkBlockState, 0, len(state))
	for k, v := range state {
		partyStates := make([]*snapshot.ProofOfWorkPartyStateForBlock, 0, len(v))
		for party, psfb := range v {
			partyStates = append(partyStates, &snapshot.ProofOfWorkPartyStateForBlock{
				Party:              party,
				ObservedDifficulty: uint64(psfb.observedDifficulty),
				SeenCount:          uint64(psfb.seenCount),
			})
		}
		sort.Slice(partyStates, func(i, j int) bool { return partyStates[i].Party < partyStates[j].Party })

		states = append(states, &snapshot.ProofOfWorkBlockState{
			BlockHeight: k,
			PartyState:  partyStates,
		})
	}
	sort.Slice(states, func(i, j int) bool { return states[i].BlockHeight < states[j].BlockHeight })
	return states
}

func (e *Engine) statesToSnapshotStates() []*snapshot.ProofOfWorkState {
	states := make([]*snapshot.ProofOfWorkState, 0, len(e.activeStates))
	for _, s := range e.activeStates {
		states = append(states, &snapshot.ProofOfWorkState{
			PowState: e.blocksState(s.blockToPartyState),
		})
	}
	return states
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

func (e *Engine) snapshotStatesToStates(activeStates []*snapshot.ProofOfWorkState) []*state {
	states := make([]*state, 0, len(activeStates))
	for _, s := range activeStates {
		currentState := &state{}
		currentState.blockToPartyState = make(map[uint64]map[string]*partyStateForBlock, len(s.PowState))
		for _, powbs := range s.PowState {
			currentState.blockToPartyState[powbs.BlockHeight] = make(map[string]*partyStateForBlock, len(powbs.PartyState))
			for _, partyState := range powbs.PartyState {
				currentState.blockToPartyState[powbs.BlockHeight][partyState.Party] = &partyStateForBlock{
					observedDifficulty: uint(partyState.ObservedDifficulty),
					seenCount:          uint(partyState.SeenCount),
				}
			}
		}
		states = append(states, currentState)
	}
	return states
}

func (e *Engine) LoadState(ctx context.Context, p *types.Payload) ([]types.StateProvider, error) {
	if e.Namespace() != p.Data.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}
	pl := p.Data.(*types.PayloadProofOfWork)
	copy(e.blockHash[:], pl.BlockHash[:ringSize])
	copy(e.blockHeight[:], pl.BlockHeight[:ringSize])
	e.heightToTx = pl.HeightToTx
	e.heightToTid = pl.HeightToTid

	for k, v := range pl.HeightToNonceRef {
		refs := make([]nonceRef, 0, len(v))
		for _, ref := range v {
			refs = append(refs, nonceRef{ref.Party, ref.Nonce})
		}
		e.heightToNonceRef[k] = refs
	}

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
	for _, block := range e.heightToNonceRef {
		for _, v := range block {
			e.seenNonceRef[v] = struct{}{}
		}
	}
	e.activeParams = e.snapshotParamsToParams(pl.ActiveParams)
	e.activeStates = e.snapshotStatesToStates(pl.ActiveStates)
	e.lastPruningBlock = pl.LastPruningBlock
	return nil, nil
}
