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

func (p *Party) Add(ord types.Party) {
	p.buf = append(p.buf, ord)
}

func (p *Party) Flush() error {
	copyBuf := p.buf
	p.buf = make([]types.Party, 0, len(copyBuf))
	return p.store.SaveBatch(copyBuf)
}
