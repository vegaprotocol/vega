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
	"sort"
	"time"

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
	bannedParties := map[string]int64{}
	for k, t := range e.bannedParties {
		bannedParties[k] = t.UnixNano()
	}

	payloadProofOfWork := &types.PayloadProofOfWork{
		BlockHeight:      e.blockHeight[:ringSize],
		BlockHash:        e.blockHash[:ringSize],
		HeightToTx:       e.heightToTx,
		HeightToTid:      e.heightToTid,
		BannedParties:    bannedParties,
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
	e.bannedParties = make(map[string]time.Time, len(pl.BannedParties))
	for k, v := range pl.BannedParties {
		e.bannedParties[k] = time.Unix(0, v)
	}
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
	e.activeStates = e.snapshotStatesToStates(pl.ActiveStates)
	e.lastPruningBlock = pl.LastPruningBlock
	return nil, nil
}
