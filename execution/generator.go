package execution

import (
	"fmt"

	"code.vegaprotocol.io/vega/types"
)

// IDgenerator no mutex required, markets work deterministically, and sequentially
type IDgenerator struct {
	batches   uint64
	orders    uint64
	proposals uint64
}

// NewIDGen returns an IDgenerator, and is used to abstract this type.
func NewIDGen() *IDgenerator {
	return &IDgenerator{}
}

// NewBatch ...
func (i *IDgenerator) NewBatch() {
	i.batches++
}

// SetID sets the ID on an order, and increments total order count
func (i *IDgenerator) SetID(o *types.Order) {
	i.orders++
	o.Id = fmt.Sprintf("V%010d-%010d", i.batches, i.orders)
}

// SetProposalID sets proposal ID and increments total proposal count
func (i *IDgenerator) SetProposalID(p *types.Proposal) {
	i.proposals++
	p.Id = fmt.Sprintf("P%010d-%010d", i.batches, i.proposals)
}
