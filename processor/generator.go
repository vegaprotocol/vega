package processor

import (
	"fmt"

	types "code.vegaprotocol.io/vega/proto"
)

// IDgenerator no mutex required, markets work deterministically, and sequentially
type IDgenerator struct {
	batches   uint64
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

// SetProposalID sets proposal ID and incrememts total proposal count
func (i *IDgenerator) SetProposalID(p *types.Proposal) {
	i.proposals++
	p.ID = fmt.Sprintf("P%010d-%010d", i.batches, i.proposals)
}
