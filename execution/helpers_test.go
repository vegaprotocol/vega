package execution_test

import (
	"code.vegaprotocol.io/vega/types"
	"github.com/imdario/mergo"
)

type OrderTemplate types.Order

func (tpl OrderTemplate) New(dst types.Order) *types.Order {
	src := types.Order(tpl)
	if err := mergo.Merge(&dst, &src); err != nil {
		panic(err)
	}

	return &dst
}
