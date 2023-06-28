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
