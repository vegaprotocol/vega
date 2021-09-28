package delegation

import (
	"context"
	"sort"
	"strings"

	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
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
		// Auto:    e.getAuto(),
	}
	return proto.Marshal(data.IntoProto())
}

func (e *Engine) Load(ctx context.Context, rawdata []byte) error {
	cp := &snapshot.Delegate{}
	if err := proto.Unmarshal(rawdata, cp); err != nil {
		return err
	}
	data := types.NewDelegationCPFromProto(cp)
	// reset state
	e.partyDelegationState = map[string]*partyDelegation{}
	e.nodeDelegationState = map[string]*validatorDelegation{}
	e.setActive(ctx, data.Active)
	e.pendingState = map[uint64]map[string]*pendingPartyDelegation{}
	e.setPending(ctx, data.Pending)
	// e.autoDelegationMode = map[string]struct{}{}
	// e.setAuto(cpData.Auto)
	return nil
}

// @TODO we probably need the context here
func (e *Engine) setActive(ctx context.Context, entries []*types.DelegationEntry) {
	nodes := []string{}
	nodeMap := map[string]struct{}{}
	for _, de := range entries {
		// add to party state
		ps, ok := e.partyDelegationState[de.Party]
		if !ok {
			ps = &partyDelegation{
				party:          de.Party,
				nodeToAmount:   map[string]*num.Uint{},
				totalDelegated: num.Zero(),
			}
			e.partyDelegationState[de.Party] = ps
		}
		if _, ok := nodeMap[de.Node]; !ok {
			nodeMap[de.Node] = struct{}{}
			nodes = append(nodes, de.Node)
		}
		ps.totalDelegated.AddSum(de.Amount)
		ps.nodeToAmount[de.Node] = de.Amount.Clone()
		// add to node state
		ns, ok := e.nodeDelegationState[de.Node]
		if !ok {
			ns = &validatorDelegation{
				nodeID:         de.Node,
				partyToAmount:  map[string]*num.Uint{},
				totalDelegated: num.Zero(),
			}
			e.nodeDelegationState[de.Node] = ns
		}
		ns.totalDelegated.AddSum(de.Amount)
		ns.partyToAmount[de.Party] = de.Amount.Clone()
	}
	sort.Strings(nodes)
	// now that we've fully restored the state, let's iterate over the parties in the same order again, and send events
	// cap is nr of parties * num of nodes
	evts := make([]events.Event, 0, len(entries)*len(nodes))
	for _, de := range entries {
		// this will always work
		ps := e.partyDelegationState[de.Party]
		for _, n := range nodes {
			amt, ok := ps.nodeToAmount[n]
			if !ok {
				amt = num.Zero()
			}
			evts = append(evts, events.NewDelegationBalance(ctx, de.Party, n, amt, num.NewUint(de.EpochSeq).String()))
		}
	}
	if len(evts) > 0 {
		e.broker.SendBatch(evts)
	}
}

func (e *Engine) getActive() []*types.DelegationEntry {
	active := make([]*types.DelegationEntry, 0, len(e.partyDelegationState)*len(e.nodeDelegationState)) // number of nodes x number of parties should be max
	// iterate over parties
	for p, ds := range e.partyDelegationState {
		for n, amt := range ds.nodeToAmount {
			active = append(active, &types.DelegationEntry{
				Party:    p,
				Node:     n,
				Amount:   amt.Clone(),
				EpochSeq: e.currentEpoch.Seq,
			})
		}
	}

	// sort the slice
	e.sortActive(active)

	return active
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

// func (e *Engine) getAuto() []string {
// 	auto := make([]string, 0, len(e.autoDelegationMode))
// 	for p := range e.autoDelegationMode {
// 		auto = append(auto, p)
// 	}
// 	sort.Strings(auto)
// 	return auto
// }

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

func (e *Engine) getPending() []*types.DelegationEntry {
	// approx. cap: undelegate nr of epoch keys * nr of parties
	// this is a worst-case scenario for sure
	pending := make([]*types.DelegationEntry, 0, len(e.pendingState)*len(e.partyDelegationState))
	for es, pp := range e.pendingState {
		for p, pds := range pp {
			// add amounts to delegate first
			for n, amt := range pds.nodeToDelegateAmount {
				pending = append(pending, &types.DelegationEntry{
					Party:    p,
					Node:     n,
					Amount:   amt.Clone(),
					EpochSeq: es,
				})
			}

			// now amounts to undelegate
			for n, amt := range pds.nodeToUndelegateAmount {
				pending = append(pending, &types.DelegationEntry{
					Party:      p,
					Node:       n,
					Amount:     amt.Clone(),
					Undelegate: true,
					EpochSeq:   es,
				})
			}
		}
	}

	e.sortPending(pending)

	return pending
}

// func (e *Engine) setAuto(parties []string) {
// 	for _, p := range parties {
// 		e.autoDelegationMode[p] = struct{}{}
// 	}
// }

func (e *Engine) setPending(ctx context.Context, entries []*types.DelegationEntry) {
	epochs := make([]uint64, 0, len(entries))
	var parties []string
	seenNodes := map[string]struct{}{}
	nodes := []string{}
	for _, pe := range entries {
		// check epoch entry
		ee, ok := e.pendingState[pe.EpochSeq]
		if !ok {
			ee = map[string]*pendingPartyDelegation{}
			epochs = append(epochs, pe.EpochSeq)
			e.pendingState[pe.EpochSeq] = ee
		}
		// check pending party entry
		ppe, ok := ee[pe.Party]
		if !ok {
			// just allocate a bit more efficiently, even though this looks messy
			if len(parties) == 0 {
				parties = make([]string, 0, len(ee))
			}
			ppe = &pendingPartyDelegation{
				party:                  pe.Party,
				nodeToDelegateAmount:   map[string]*num.Uint{},
				nodeToUndelegateAmount: map[string]*num.Uint{},
				totalDelegation:        num.Zero(),
				totalUndelegation:      num.Zero(),
			}
			// this assumes only 1 epoch... otherwise duplicate events are possible
			parties = append(parties, pe.Party)
			ee[pe.Party] = ppe
		}
		if _, ok := seenNodes[pe.Node]; !ok {
			seenNodes[pe.Node] = struct{}{}
			nodes = append(nodes, pe.Node)
		}
		// delegate/undelegate?
		if pe.Undelegate {
			ppe.totalUndelegation.AddSum(pe.Amount)
			ppe.nodeToUndelegateAmount[pe.Node] = pe.Amount.Clone()
		} else {
			ppe.totalDelegation.AddSum(pe.Amount)
			ppe.nodeToDelegateAmount[pe.Node] = pe.Amount.Clone()
		}
		// just to be sure the main map is updated...
		e.pendingState[pe.EpochSeq] = ee
	}
	sort.Strings(parties)
	sort.Strings(nodes)
	sort.SliceStable(epochs, func(i, j int) bool {
		return epochs[i] < epochs[j]
	})
	evts := make([]events.Event, 0, len(parties)*len(epochs)*len(nodes))
	for _, seq := range epochs {
		for _, p := range parties {
			for _, n := range nodes {
				evts = append(evts, e.getNextEpochBalanceEvent(ctx, p, n, seq))
			}
		}
	}
	if len(evts) > 0 {
		e.broker.SendBatch(evts)
	}
}
