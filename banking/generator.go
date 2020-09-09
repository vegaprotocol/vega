package banking

import (
	"fmt"
	"math/big"
	"time"

	types "code.vegaprotocol.io/vega/proto"
)

// IDgenerator no mutex required, markets work deterministically, and sequentially
type IDgenerator struct {
	batches     uint64
	withdrawals uint64
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
func (i *IDgenerator) SetID(w *types.Withdrawal, t time.Time) *big.Int {
	i.withdrawals++
	ref := big.NewInt(int64(i.withdrawals))
	ref = ref.Add(ref, big.NewInt(t.Unix()))
	w.Id = fmt.Sprintf("W%010d-%010d", i.batches, i.withdrawals)
	w.Ref = ref.String()
	return ref
}
