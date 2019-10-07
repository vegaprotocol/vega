package execution

import (
	"fmt"

	types "code.vegaprotocol.io/vega/proto"
)

// IDgenerator no mutex required, markets work deterministically, and sequentially
type idgenerator struct {
	blocks uint64
	orders uint64
}

// we don't really need this func, but we want to abstract/obscure
// this type as much as possible
func newIDGen() *idgenerator {
	return &idgenerator{}
}

// NewBlock ...
func (i *idgenerator) newBlock() {
	i.blocks++
}

// setID - sets id on an order, and increments total order count
func (i *idgenerator) setID(o *types.Order) {
	i.orders++
	o.Id = fmt.Sprintf("V%010d-%010d", i.blocks, i.orders)
}
