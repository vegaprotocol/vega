package stoporders

import (
	"fmt"
	"strings"

	"github.com/google/btree"
)

func dumpTree[T fmt.Stringer](tree *btree.BTreeG[T]) string {
	var out []string
	tree.Ascend(func(item T) bool {
		out = append(out, fmt.Sprintf("(%s)", item.String()))
		return true
	})
	return strings.Join(out, ",")

}
