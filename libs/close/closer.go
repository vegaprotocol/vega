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

package close

type Closer struct {
	closeFns []func()
}

// Add adds a function to call during call to CloseAll.
func (c *Closer) Add(closeFn func()) {
	c.closeFns = append(c.closeFns, closeFn)
}

// CloseAll calls all close functions in reverse order.
// Higher level-components should be closed first, but are usually instantiated
// last (and, thus, added later to the closer), hence the reverse order.
func (c *Closer) CloseAll() {
	for i := len(c.closeFns) - 1; i >= 0; i-- {
		c.closeFns[i]()
	}

	c.closeFns = []func(){}
}

func NewCloser() *Closer {
	return &Closer{
		closeFns: []func(){},
	}
}
