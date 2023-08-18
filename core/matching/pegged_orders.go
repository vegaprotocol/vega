package matching

import "github.com/pkg/errors"

var ErrUnknownPeggedOrderID = errors.New("unknow pegged order")

type peggedOrders struct {
	ids []string
}

func newPeggedOrders() *peggedOrders {
	return &peggedOrders{
		ids: []string{},
	}
}

func (p *peggedOrders) Clear() {
	p.ids = []string{}
}

func (p *peggedOrders) Add(id string) {
	p.ids = append(p.ids, id)
}

func (p *peggedOrders) Delete(id string) error {
	idx := -1
	for i, v := range p.ids {
		if v == id {
			idx = i
			break
		}
	}

	if idx == -1 {
		return ErrUnknownPeggedOrderID
	}

	if idx < len(p.ids)-1 {
		copy(p.ids[idx:], p.ids[idx+1:])
	}
	p.ids[len(p.ids)-1] = ""
	p.ids = p.ids[:len(p.ids)-1]

	return nil
}

func (p *peggedOrders) Exists(id string) bool {
	for _, v := range p.ids {
		if v == id {
			return true
		}
	}

	return false
}

func (p *peggedOrders) Iter() []string {
	return p.ids
}

func (p *peggedOrders) Len() int {
	return len(p.ids)
}
