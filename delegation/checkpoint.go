package delegation

import (
	"context"
	"sort"
	"strings"

	checkpoint "code.vegaprotocol.io/protos/vega/checkpoint/v1"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/golang/protobuf/proto"
)

func (e *Engine) Name() types.CheckpointName {
	return types.DelegationCheckpoint
}

func (e *Engine) Checkpoint() ([]byte, error) {
	data := &types.DelegateCP{
		Active:  e.getActive(),
		Pending: e.getPending(),
		Auto:    e.getAuto(),
	}
	return proto.Marshal(data.IntoProto())
}

func (e *Engine) Load(ctx context.Context, data []byte) error {
	cp := &checkpoint.Delegate{}
	if err := proto.Unmarshal(data, cp); err != nil {
		return err
	}
	cpData := types.NewDelegationCPFromProto(cp)
	// reset state
	e.partyDelegationState = map[string]*partyDelegation{}
	e.nextPartyDelegationState = map[string]*partyDelegation{}
	e.setActive(ctx, cpData.Active)
	e.setPending(ctx, cpData.Pending)

	e.autoDelegationMode = map[string]struct{}{}
	e.setAuto(cpData.Auto)

	return nil
}

func (e *Engine) delegationStateFromDelegationEntry(ctx context.Context, delegationState map[string]*partyDelegation, entries []*types.DelegationEntry) {
	// each entry results in a delegation event
	evts := make([]events.Event, 0, len(entries))
	// bit silly, but has to be more efficient than num.NewUint(EpochSeq).String() every time
	epochStr := map[uint64]string{}
	for _, de := range entries {
		// add to party state
		ps, ok := delegationState[de.Party]
		if !ok {
			ps = &partyDelegation{
				party:          de.Party,
				nodeToAmount:   map[string]*num.Uint{},
				totalDelegated: num.Zero(),
			}
			delegationState[de.Party] = ps
		}
		ps.totalDelegated.AddSum(de.Amount)
		ps.nodeToAmount[de.Node] = de.Amount.Clone()
		eStr, ok := epochStr[de.EpochSeq]
		if !ok {
			eStr = num.NewUint(de.EpochSeq).String()
			epochStr[de.EpochSeq] = eStr
		}
		evts = append(evts, events.NewDelegationBalance(ctx, de.Party, de.Node, de.Amount, eStr))
	}
	if len(evts) > 0 {
		e.broker.SendBatch(evts)
	}
}

func (e *Engine) setActive(ctx context.Context, entries []*types.DelegationEntry) {
	e.delegationStateFromDelegationEntry(ctx, e.partyDelegationState, entries)
}

func (e *Engine) delegationStateToDelegationEntry(delegationState map[string]*partyDelegation, epochSeq uint64) []*types.DelegationEntry {
	slice := []*types.DelegationEntry{}
	// iterate over parties
	for p, ds := range delegationState {
		for n, amt := range ds.nodeToAmount {
			slice = append(slice, &types.DelegationEntry{
				Party:    p,
				Node:     n,
				Amount:   amt.Clone(),
				EpochSeq: epochSeq,
			})
		}
	}

	// sort the slice
	e.sortActive(slice)

	return slice
}

func (e *Engine) getActive() []*types.DelegationEntry {
	return e.delegationStateToDelegationEntry(e.partyDelegationState, e.currentEpoch.Seq)
}

func (e *Engine) sortActive(active []*types.DelegationEntry) {
	sort.SliceStable(active, func(i, j int) bool {
		switch strings.Compare(active[i].Party, active[j].Party) {
		case -1:
			return true
		case 1:
			return false
		}

		return active[i].Node < active[j].Node
	})
}

func (e *Engine) getAuto() []string {
	auto := make([]string, 0, len(e.autoDelegationMode))
	for p := range e.autoDelegationMode {
		auto = append(auto, p)
	}
	sort.Strings(auto)
	return auto
}

func (e *Engine) getPending() []*types.DelegationEntry {
	return e.delegationStateToDelegationEntry(e.nextPartyDelegationState, e.currentEpoch.Seq+1)
}

func (e *Engine) setAuto(parties []string) {
	for _, p := range parties {
		e.autoDelegationMode[p] = struct{}{}
	}
}

func (e *Engine) setPending(ctx context.Context, entries []*types.DelegationEntry) {
	e.delegationStateFromDelegationEntry(ctx, e.nextPartyDelegationState, entries)
}
