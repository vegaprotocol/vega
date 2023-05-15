package execution

import "code.vegaprotocol.io/vega/libs/num"

func priceToMarketPrecision(price *num.Uint, priceFactor *num.Uint) *num.Uint {
	// we assume the price is cloned correctly already
	return price.Div(price, priceFactor)
}
