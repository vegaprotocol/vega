package execution

import (
	"fmt"
	"time"

	types "code.vegaprotocol.io/vega/proto"
)

// no mutex required, markets work deterministically, and sequentially
type idgenerator struct {
	now    time.Time
	blocks uint64
	orders uint64
}

// we don't really need this func, but we want to abstract/obscure
// this type as much as possible
func newIDGen() *idgenerator {
	return &idgenerator{}
}

// updateTime - set new block time, increases block count, too
func (i *idgenerator) updateTime(t time.Time) {
	i.now = t
	i.blocks++
}

// setID - sets id on an order, and increments total order count
func (i *idgenerator) setID(o *types.Order) {
	i.orders++
	o.Id = fmt.Sprintf("V%010d-%010d", i.blocks, i.orders)
	// just make sure this is set
	if o.CreatedAt == 0 {
		o.CreatedAt = i.now.UnixNano()
	}
}
