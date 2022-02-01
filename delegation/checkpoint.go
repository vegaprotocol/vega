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
		Pending: e.getPendingBackwardCompatible(),
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
	e.setPendingBackwardCompatible(ctx, cpData.Pending)

	e.autoDelegationMode = map[string]struct{}{}
	e.setAuto(cpData.Auto)

	return nil
}

func (e *Engine) delegationStateFromDelegationEntry(ctx context.Context, delegationState map[string]*partyDelegation, entries []*types.DelegationEntry, epochSeq string) {
	// each entry results in a delegation event
	evts := make([]events.Event, 0, len(entries))
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

		evts = append(evts, events.NewDelegationBalance(ctx, de.Party, de.Node, de.Amount, epochSeq))
	}
	if len(evts) > 0 {
		e.broker.SendBatch(evts)
	}
}

func (e *Engine) setActive(ctx context.Context, entries []*types.DelegationEntry) {
	e.delegationStateFromDelegationEntry(ctx, e.partyDelegationState, entries, num.NewUint(e.currentEpoch.Seq).String())
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

func (e *Engine) getPendingNew() []*types.DelegationEntry {
	return e.delegationStateToDelegationEntry(e.nextPartyDelegationState, e.currentEpoch.Seq+1)
}

// getPendingBackwardCompatible is calculating deltas based on next epoch balances to be saved in the checkpoint.
// this is because for backward compatibility we continue to save deltas rather than balances in the checkpoint for pending (i.e. next epoch's delegations).
func (e *Engine) getPendingBackwardCompatible() []*types.DelegationEntry {
	des := []*types.DelegationEntry{}
	for party, state := range e.nextPartyDelegationState {
		currState, ok := e.partyDelegationState[party]
		if !ok {
			for node, amt := range state.nodeToAmount {
				des = append(des, &types.DelegationEntry{Party: party, Node: node, Amount: amt.Clone(), Undelegate: false, EpochSeq: e.currentEpoch.Seq + 1})
			}
			continue
		}
		for node, amt := range state.nodeToAmount {
			currNodeAmt, ok := currState.nodeToAmount[node]
			if !ok {
				des = append(des, &types.DelegationEntry{Party: party, Node: node, Amount: amt.Clone(), Undelegate: false, EpochSeq: e.currentEpoch.Seq + 1})
			} else {
				if amt.GT(currNodeAmt) {
					des = append(des, &types.DelegationEntry{Party: party, Node: node, Amount: num.Zero().Sub(amt, currNodeAmt), Undelegate: false, EpochSeq: e.currentEpoch.Seq + 1})
				} else if amt.LT(currNodeAmt) {
					des = append(des, &types.DelegationEntry{Party: party, Node: node, Amount: num.Zero().Sub(currNodeAmt, amt), Undelegate: true, EpochSeq: e.currentEpoch.Seq + 1})
				}
			}
		}
		// handle nominations that were removed completely
		for currNode, currAmt := range currState.nodeToAmount {
			if _, ok := state.nodeToAmount[currNode]; !ok {
				des = append(des, &types.DelegationEntry{Party: party, Node: currNode, Amount: currAmt.Clone(), Undelegate: true, EpochSeq: e.currentEpoch.Seq + 1})
			}
		}
	}

	// handle nominations from parties that were completed cleared
	for party, state := range e.partyDelegationState {
		if _, ok := e.nextPartyDelegationState[party]; !ok {
			for node, amt := range state.nodeToAmount {
				des = append(des, &types.DelegationEntry{Party: party, Node: node, Amount: amt.Clone(), Undelegate: true, EpochSeq: e.currentEpoch.Seq + 1})
			}
		}
	}

	e.sortPending(des)
	return des
}

func (e *Engine) sortPending(pending []*types.DelegationEntry) {
	sort.SliceStable(pending, func(i, j int) bool {
		pi, pj := pending[i], pending[j]

		switch strings.Compare(pi.Party, pj.Party) {
		case -1:
			return true
		case 1:
			return false
		}

		switch strings.Compare(pi.Node, pj.Node) {
		case -1:
			return true
		case 1:
			return false
		}

		return pi.EpochSeq < pj.EpochSeq
	})
}

func (e *Engine) setAuto(parties []string) {
	for _, p := range parties {
		e.autoDelegationMode[p] = struct{}{}
	}
}

func (e *Engine) setPendingNew(ctx context.Context, entries []*types.DelegationEntry) {
	e.delegationStateFromDelegationEntry(ctx, e.nextPartyDelegationState, entries, num.NewUint(e.currentEpoch.Seq+1).String())
}

// setPendingBackwardCompatible is taking deltas from the checkpoint and calculate from them the associated balance for the next epoch
// populating nextPartyDelegationState.
// NB: the event emitted are based on the *current* epoch in play rather than on the meaningless epoch from the DelegationEntry.
func (e *Engine) setPendingBackwardCompatible(ctx context.Context, entries []*types.DelegationEntry) {
	// first initialise the state with the current state
	for party, pds := range e.partyDelegationState {
		e.nextPartyDelegationState[party] = &partyDelegation{
			party:          party,
			nodeToAmount:   map[string]*num.Uint{},
			totalDelegated: pds.totalDelegated.Clone(),
		}
		for node, amt := range pds.nodeToAmount {
			e.nextPartyDelegationState[party].nodeToAmount[node] = amt.Clone()
		}
	}

	for _, de := range entries {
		// add to party state
		ps, ok := e.nextPartyDelegationState[de.Party]
		if !ok {
			ps = &partyDelegation{
				party:          de.Party,
				nodeToAmount:   map[string]*num.Uint{},
				totalDelegated: num.Zero(),
			}
			e.nextPartyDelegationState[de.Party] = ps
		}

		if !de.Undelegate {
			ps.totalDelegated.AddSum(de.Amount)
			if _, ok := ps.nodeToAmount[de.Node]; !ok {
				ps.nodeToAmount[de.Node] = de.Amount.Clone()
			} else {
				ps.nodeToAmount[de.Node].AddSum(de.Amount)
			}
		} else {
			if _, ok := ps.nodeToAmount[de.Node]; ok {
				amt := num.Min(ps.nodeToAmount[de.Node], de.Amount)
				ps.nodeToAmount[de.Node].Sub(ps.nodeToAmount[de.Node], amt)
				ps.totalDelegated.Sub(ps.totalDelegated, amt)
			}
		}
	}

	epoch := num.NewUint(e.currentEpoch.Seq + 1).String()
	evts := []events.Event{}
	parties := e.sortParties(e.nextPartyDelegationState)
	for _, party := range parties {
		nodes := e.sortNodes(e.nextPartyDelegationState[party].nodeToAmount)
		for _, node := range nodes {
			balance := e.nextPartyDelegationState[party].nodeToAmount[node]
			evts = append(evts, events.NewDelegationBalance(ctx, party, node, balance, epoch))
			if balance.IsZero() {
				delete(e.nextPartyDelegationState[party].nodeToAmount, node)
			}
		}
		if e.nextPartyDelegationState[party].totalDelegated.IsZero() {
			delete(e.nextPartyDelegationState, party)
		}
	}

	if len(evts) > 0 {
		e.broker.SendBatch(evts)
	}
}
