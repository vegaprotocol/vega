package execution_test

import (
	types "code.vegaprotocol.io/vega/proto"
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
