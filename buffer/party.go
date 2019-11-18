package buffer

import types "code.vegaprotocol.io/vega/proto"

type PartyStore interface {
	SaveBatch(order []types.Party) error
}

type Party struct {
	store PartyStore
	buf   []types.Party
}

func NewParty(store PartyStore) *Party {
	return &Party{
		store: store,
		buf:   []types.Party{},
	}
}

func (o *Party) Add(ord types.Party) {
	o.buf = append(o.buf, ord)
}

func (o *Party) Flush() error {
	copyBuf := o.buf
	o.buf = make([]types.Party, 0, len(copyBuf))
	return o.store.SaveBatch(copyBuf)
}
