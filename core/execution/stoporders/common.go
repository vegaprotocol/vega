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

package stoporders

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/btree"
)

var (
	ErrNoPriceToOffset   = errors.New("no price to offset")
	ErrStopOrderNotFound = errors.New("stop order not found")
	ErrPriceNotFound     = errors.New("price not found")
	ErrOrderNotFound     = errors.New("order not found")
)

func dumpTree[T fmt.Stringer](tree *btree.BTreeG[T]) string {
	var out []string
	tree.Ascend(func(item T) bool {
		out = append(out, fmt.Sprintf("(%s)", item.String()))
		return true
	})

	return strings.Join(out, ",")
}
