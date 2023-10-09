// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
