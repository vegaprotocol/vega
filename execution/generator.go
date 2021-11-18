package execution

import (
	"fmt"

	"code.vegaprotocol.io/vega/types"
)

// IDgenerator no mutex required, markets work deterministically, and sequentially.
type IDgenerator struct {
	batches   uint64
	orders    uint64
	proposals uint64

	changed bool
}

// NewIDGen returns an IDgenerator, and is used to abstract this type.
func NewIDGen() *IDgenerator {
	return &IDgenerator{
		changed: true,
	}
}

// NewBatch ...
func (i *IDgenerator) NewBatch() {
	i.batches++
	i.changed = true
}

// SetID sets the ID on an order, and increments total order count.
func (i *IDgenerator) SetID(o *types.Order) {
	i.orders++
	o.ID = fmt.Sprintf("V%010d-%010d", i.batches, i.orders)
	i.changed = true
}

// SetProposalID sets proposal ID and increments total proposal count.
func (i *IDgenerator) SetProposalID(p *types.Proposal) {
	i.proposals++
	p.ID = fmt.Sprintf("P%010d-%010d", i.batches, i.proposals)
	i.changed = true
}

func (i *IDgenerator) Changed() bool {
	return i.changed
}

func (i *IDgenerator) SnapshotCreated() {
	i.changed = false
}
