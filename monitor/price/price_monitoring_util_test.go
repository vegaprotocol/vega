package price

import (
	"code.vegaprotocol.io/vega/types/num"
)

func (e *Engine) UpdateTestFactors(down, up []num.Decimal) {
	e.updateFactors(down, up)
}
