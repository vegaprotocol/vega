package assets

import (
	"fmt"
)

// IDgenerator no mutex required, markets work deterministically, and sequentially
type IDgenerator struct {
	batches uint64
	assets  uint64
}

// NewIDGen returns an IDgenerator, and is used to abstract this type.
func NewIDGen() *IDgenerator {
	return &IDgenerator{}
}

// NewBatch ...
func (i *IDgenerator) NewBatch() {
	i.batches++
}

// SetID sets the ID on an asset, and increments total assets count
func (i *IDgenerator) NewID() string {
	i.assets++
	return fmt.Sprintf("V%010d-%010d", i.batches, i.assets)
}
