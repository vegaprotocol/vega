package delegation

import (
	"sort"
	"strings"

	checkpoint "code.vegaprotocol.io/protos/vega/checkpoint/v1"
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

func (e *Engine) Load(data []byte) error {
	cp := &checkpoint.Delegate{}
	if err := proto.Unmarshal(data, cp); err != nil {
		return err
	}
	cpData := types.NewDelegationCPFromProto(cp)
	// reset state
	e.partyDelegationState = map[string]*partyDelegation{}
	e.nodeDelegationState = map[string]*validatorDelegation{}
	e.setActive(cpData.Active)
	e.pendingState = map[uint64]map[string]*pendingPartyDelegation{}
	e.setPending(cpData.Pending)
	// e.autoDelegationMode = map[string]struct{}{}
	// e.setAuto(cpData.Auto)

	return nil
}

func (e *Engine) setActive(entries []*types.DelegationEntry) {
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

func (e *Engine) setPending(entries []*types.DelegationEntry) {
	for _, pe := range entries {
		// check epoch entry
		ee, ok := e.pendingState[pe.EpochSeq]
		if !ok {
			ee = map[string]*pendingPartyDelegation{}
			e.pendingState[pe.EpochSeq] = ee
		}
		// check pending party entry
		ppe, ok := ee[pe.Party]
		if !ok {
			ppe = &pendingPartyDelegation{
				party:                  pe.Party,
				nodeToDelegateAmount:   map[string]*num.Uint{},
				nodeToUndelegateAmount: map[string]*num.Uint{},
				totalDelegation:        num.Zero(),
				totalUndelegation:      num.Zero(),
			}
			ee[pe.Party] = ppe
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
}
