package execution

import (
	"fmt"

	types "code.vegaprotocol.io/vega/proto"
)

// IDgenerator no mutex required, markets work deterministically, and sequentially
type IDgenerator struct {
	batches uint64
	orders  uint64
}

// we don't really need this func, but we want to abstract/obscure
// this type as much as possible
func NewIDGen() *IDgenerator {
	return &IDgenerator{}
}

// NewBlock ...
func (i *IDgenerator) NewBatch() {
	i.batches++
}

// setID - sets id on an order, and increments total order count
func (i *IDgenerator) SetID(o *types.Order) {
	i.orders++
	o.Id = fmt.Sprintf("V%010d-%010d", i.batches, i.orders)
}
